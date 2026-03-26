package openapi

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/compatgate/compatgate/internal/diff"
	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/normalize"
	"github.com/compatgate/compatgate/internal/protocols"
	"github.com/compatgate/compatgate/pkg/compatgate"
	"github.com/getkin/kin-openapi/openapi3"
)

func Analyze(ctx context.Context, request compatgate.Request) (findings.Report, error) {
	baseDoc, err := parse(ctx, request.Base)
	if err != nil {
		return findings.Report{}, err
	}
	revisionDoc, err := parse(ctx, request.Revision)
	if err != nil {
		return findings.Report{}, err
	}
	result := diff.Compare(normalizeDocument(baseDoc, request.Base), normalizeDocument(revisionDoc, request.Revision))
	items := evaluate(result)
	sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
	return findings.NewReport([]findings.Protocol{findings.ProtocolOpenAPI}, request.Base, request.Revision, items), nil
}

func parse(ctx context.Context, location string) (*openapi3.T, error) {
	data, err := protocols.LoadSource(ctx, location)
	if err != nil {
		return nil, err
	}
	loader := openapi3.NewLoader()
	return loader.LoadFromData(data)
}

func normalizeDocument(doc *openapi3.T, file string) normalize.Contract {
	contract := normalize.Contract{Protocol: findings.ProtocolOpenAPI}
	if doc.Paths == nil {
		return contract
	}
	for path, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		for method, operation := range operations(item) {
			if operation == nil {
				continue
			}
			opID := fmt.Sprintf("operation:%s:%s", strings.ToUpper(method), path)
			contract.Resources = append(contract.Resources, normalize.Resource{
				Kind:       "operation",
				Name:       strings.ToUpper(method) + " " + path,
				Identifier: opID,
				Meta:       map[string]string{"path": path, "method": strings.ToUpper(method)},
				Source:     &findings.SourceLocation{File: file},
			})
			for _, parameter := range operation.Parameters {
				if parameter == nil || parameter.Value == nil {
					continue
				}
				param := parameter.Value
				contract.Resources = append(contract.Resources, normalize.Resource{
					Kind:       "parameter",
					Name:       param.Name,
					Parent:     opID,
					Identifier: fmt.Sprintf("parameter:%s:%s:%s", opID, param.In, param.Name),
					Required:   param.Required,
					Type:       schemaType(param.Schema),
					EnumValues: schemaEnum(param.Schema),
					Meta:       map[string]string{"in": param.In},
					Source:     &findings.SourceLocation{File: file},
				})
			}
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				for _, schemaRef := range bodySchemas(operation.RequestBody.Value.Content) {
					contract.Resources = append(contract.Resources, collectSchemaResources("request-field", opID, schemaRef, file)...)
				}
			}
			if operation.Responses != nil {
				for status, response := range operation.Responses.Map() {
					if !strings.HasPrefix(status, "2") && status != "default" {
						continue
					}
					if response == nil || response.Value == nil {
						continue
					}
					parent := opID + ":response:" + status
					for _, schemaRef := range bodySchemas(response.Value.Content) {
						contract.Resources = append(contract.Resources, collectSchemaResources("response-field", parent, schemaRef, file)...)
					}
				}
			}
		}
	}
	return contract
}

func operations(item *openapi3.PathItem) map[string]*openapi3.Operation {
	return map[string]*openapi3.Operation{
		"get":     item.Get,
		"put":     item.Put,
		"post":    item.Post,
		"delete":  item.Delete,
		"patch":   item.Patch,
		"options": item.Options,
		"head":    item.Head,
	}
}

func bodySchemas(content openapi3.Content) []*openapi3.SchemaRef {
	for _, key := range []string{"application/json", "application/*+json"} {
		if media := content.Get(key); media != nil && media.Schema != nil {
			return []*openapi3.SchemaRef{media.Schema}
		}
	}
	items := make([]*openapi3.SchemaRef, 0)
	for _, media := range content {
		if media != nil && media.Schema != nil {
			items = append(items, media.Schema)
		}
	}
	return items
}

func collectSchemaResources(kind string, parent string, schemaRef *openapi3.SchemaRef, file string) []normalize.Resource {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil
	}
	return collectSchemaResourcesRecursive(kind, parent, schemaRef, file)
}

func collectSchemaResourcesRecursive(kind string, parent string, schemaRef *openapi3.SchemaRef, file string) []normalize.Resource {
	if schemaRef == nil || schemaRef.Value == nil {
		return nil
	}
	schema := schemaRef.Value
	items := make([]normalize.Resource, 0)
	requiredSet := make(map[string]bool, len(schema.Required))
	for _, name := range schema.Required {
		requiredSet[name] = true
	}
	keys := make([]string, 0, len(schema.Properties))
	for key := range schema.Properties {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, propertyName := range keys {
		property := schema.Properties[propertyName]
		resourceID := fmt.Sprintf("%s:%s", parent, propertyName)
		resource := normalize.Resource{
			Kind:       kind,
			Name:       propertyName,
			Parent:     parent,
			Identifier: resourceID,
			Required:   requiredSet[propertyName],
			Type:       schemaType(property),
			EnumValues: schemaEnum(property),
			Source:     &findings.SourceLocation{File: file},
		}
		items = append(items, resource)
		items = append(items, collectSchemaResourcesRecursive(kind, resourceID, property, file)...)
	}
	if schema.Items != nil {
		items = append(items, collectSchemaResourcesRecursive(kind, parent+"[]", schema.Items, file)...)
	}
	return items
}

func schemaType(ref *openapi3.SchemaRef) string {
	if ref == nil || ref.Value == nil {
		return ""
	}
	schema := ref.Value
	if schema.Type != nil && len(*schema.Type) > 0 {
		return strings.Join([]string(*schema.Type), "|")
	}
	if ref.Ref != "" {
		return ref.Ref
	}
	return ""
}

func schemaEnum(ref *openapi3.SchemaRef) []string {
	if ref == nil || ref.Value == nil || len(ref.Value.Enum) == 0 {
		return nil
	}
	values := make([]string, 0, len(ref.Value.Enum))
	for _, raw := range ref.Value.Enum {
		values = append(values, fmt.Sprintf("%v", raw))
	}
	slices.Sort(values)
	return values
}

func evaluate(result diff.ContractDiff) []findings.Finding {
	items := make([]findings.Finding, 0)
	for _, removed := range result.Removed {
		switch removed.Kind {
		case "operation":
			items = append(items, finding("openapi.operation.removed", removed, "operation removed"))
		case "parameter":
			items = append(items, finding("openapi.parameter.removed", removed, "parameter removed"))
		case "response-field":
			items = append(items, finding("openapi.response_field.removed", removed, "response field removed"))
		}
	}
	for _, added := range result.Added {
		switch added.Kind {
		case "parameter":
			if added.Required {
				items = append(items, finding("openapi.parameter.required_added", added, "required parameter added"))
			}
		case "request-field":
			if added.Required {
				items = append(items, finding("openapi.request_field.required_added", added, "required request field added"))
			}
		}
	}
	for _, changed := range result.Changed {
		if changed.Before == nil || changed.After == nil {
			continue
		}
		before := *changed.Before
		after := *changed.After
		switch after.Kind {
		case "parameter":
			if !before.Required && after.Required {
				items = append(items, changedFinding("openapi.parameter.required_tightened", before, after, "parameter became required"))
			}
			if before.Type != after.Type && before.Type != "" && after.Type != "" {
				items = append(items, changedFinding("openapi.parameter.type_changed", before, after, "parameter type changed"))
			}
			if protocols.EnumShrunk(before.EnumValues, after.EnumValues) {
				items = append(items, changedFinding("openapi.parameter.enum_shrunk", before, after, "parameter enum narrowed"))
			}
		case "request-field":
			if !before.Required && after.Required {
				items = append(items, changedFinding("openapi.request_field.required_tightened", before, after, "request field became required"))
			}
			if before.Type != after.Type && before.Type != "" && after.Type != "" {
				items = append(items, changedFinding("openapi.request_field.type_changed", before, after, "request field type changed"))
			}
		case "response-field":
			if before.Type != after.Type && before.Type != "" && after.Type != "" {
				items = append(items, changedFinding("openapi.response_field.type_changed", before, after, "response field type changed"))
			}
			if protocols.EnumShrunk(before.EnumValues, after.EnumValues) {
				items = append(items, changedFinding("openapi.response_field.enum_shrunk", before, after, "response field enum narrowed"))
			}
		}
	}
	return items
}

func finding(ruleID string, resource normalize.Resource, message string) findings.Finding {
	return protocols.Finding(findings.ProtocolOpenAPI, ruleID, findings.SeverityError, true, resource, message, nil, nil)
}

func changedFinding(ruleID string, before normalize.Resource, after normalize.Resource, message string) findings.Finding {
	return protocols.ChangedFinding(findings.ProtocolOpenAPI, ruleID, findings.SeverityError, true, before, after, message)
}
