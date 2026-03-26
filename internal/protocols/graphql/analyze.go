package graphql

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/compatgate/compatgate/internal/diff"
	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/normalize"
	"github.com/compatgate/compatgate/internal/protocols"
	"github.com/compatgate/compatgate/pkg/compatgate"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func Analyze(ctx context.Context, request compatgate.Request) (findings.Report, error) {
	baseSchema, err := parse(ctx, request.Base)
	if err != nil {
		return findings.Report{}, err
	}
	revisionSchema, err := parse(ctx, request.Revision)
	if err != nil {
		return findings.Report{}, err
	}
	result := diff.Compare(normalizeSchema(baseSchema, request.Base), normalizeSchema(revisionSchema, request.Revision))
	items := evaluate(result)
	sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
	return findings.NewReport([]findings.Protocol{findings.ProtocolGraphQL}, request.Base, request.Revision, items), nil
}

func parse(ctx context.Context, location string) (*ast.Schema, error) {
	data, err := protocols.LoadSource(ctx, location)
	if err != nil {
		return nil, err
	}
	return gqlparser.LoadSchema(&ast.Source{Name: location, Input: string(data)})
}

func normalizeSchema(schema *ast.Schema, file string) normalize.Contract {
	contract := normalize.Contract{Protocol: findings.ProtocolGraphQL}
	for name, definition := range schema.Types {
		if strings.HasPrefix(name, "__") || definition == nil || definition.BuiltIn {
			continue
		}
		typeID := "type:" + name
		contract.Resources = append(contract.Resources, normalize.Resource{
			Kind:       "type",
			Name:       name,
			Identifier: typeID,
			Type:       string(definition.Kind),
			Meta:       map[string]string{"kind": string(definition.Kind)},
			Source:     &findings.SourceLocation{File: file},
		})
		for _, field := range definition.Fields {
			fieldID := fmt.Sprintf("field:%s:%s", name, field.Name)
			contract.Resources = append(contract.Resources, normalize.Resource{
				Kind:       "field",
				Name:       field.Name,
				Parent:     name,
				Identifier: fieldID,
				Required:   field.Type.NonNull,
				Type:       field.Type.String(),
				Meta:       map[string]string{"type_kind": string(definition.Kind)},
				Source:     &findings.SourceLocation{File: file},
			})
			for _, argument := range field.Arguments {
				contract.Resources = append(contract.Resources, normalize.Resource{
					Kind:       "argument",
					Name:       argument.Name,
					Parent:     fieldID,
					Identifier: fmt.Sprintf("argument:%s:%s", fieldID, argument.Name),
					Required:   argument.Type.NonNull,
					Type:       argument.Type.String(),
					Source:     &findings.SourceLocation{File: file},
				})
			}
		}
		for _, enumValue := range definition.EnumValues {
			contract.Resources = append(contract.Resources, normalize.Resource{
				Kind:       "enum-value",
				Name:       enumValue.Name,
				Parent:     name,
				Identifier: fmt.Sprintf("enum:%s:%s", name, enumValue.Name),
				Source:     &findings.SourceLocation{File: file},
			})
		}
	}
	return contract
}

func evaluate(result diff.ContractDiff) []findings.Finding {
	items := make([]findings.Finding, 0)
	for _, removed := range result.Removed {
		switch removed.Kind {
		case "type":
			items = append(items, finding("graphql.type.removed", removed, "type removed"))
		case "field":
			items = append(items, finding("graphql.field.removed", removed, "field removed"))
		case "argument":
			items = append(items, finding("graphql.argument.removed", removed, "argument removed"))
		case "enum-value":
			items = append(items, finding("graphql.enum_value.removed", removed, "enum value removed"))
		}
	}
	for _, added := range result.Added {
		if added.Kind == "field" && added.Meta["type_kind"] == "INPUT_OBJECT" && added.Required {
			items = append(items, finding("graphql.input_field.required_added", added, "required input field added"))
		}
		if added.Kind == "argument" && added.Required {
			items = append(items, finding("graphql.argument.required_added", added, "required argument added"))
		}
	}
	for _, changed := range result.Changed {
		before, after := *changed.Before, *changed.After
		switch after.Kind {
		case "field":
			if !before.Required && after.Required {
				items = append(items, changedFinding("graphql.field.nullable_to_nonnull", before, after, "field became non-null"))
			}
			if before.Type != after.Type {
				items = append(items, changedFinding("graphql.field.type_changed", before, after, "field return type changed"))
			}
		case "argument":
			if !before.Required && after.Required {
				items = append(items, changedFinding("graphql.argument.nullable_to_nonnull", before, after, "argument became required"))
			}
			if before.Type != after.Type {
				items = append(items, changedFinding("graphql.argument.type_changed", before, after, "argument type changed"))
			}
		}
	}
	return items
}

func finding(ruleID string, resource normalize.Resource, message string) findings.Finding {
	return protocols.Finding(findings.ProtocolGraphQL, ruleID, findings.SeverityError, true, resource, message, nil, nil)
}

func changedFinding(ruleID string, before normalize.Resource, after normalize.Resource, message string) findings.Finding {
	return protocols.ChangedFinding(findings.ProtocolGraphQL, ruleID, findings.SeverityError, true, before, after, message)
}
