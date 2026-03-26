package protobuf

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	protoparser "github.com/emicklei/proto"

	"github.com/compatgate/compatgate/internal/diff"
	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/normalize"
	"github.com/compatgate/compatgate/internal/protocols"
	"github.com/compatgate/compatgate/pkg/compatgate"
)

func Analyze(ctx context.Context, request compatgate.Request) (findings.Report, error) {
	base, err := loadContract(ctx, request.Base)
	if err != nil {
		return findings.Report{}, err
	}
	revision, err := loadContract(ctx, request.Revision)
	if err != nil {
		return findings.Report{}, err
	}
	result := diff.Compare(base, revision)
	items := buildFindings(base, revision, result)
	sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
	return findings.NewReport([]findings.Protocol{findings.ProtocolGRPC}, request.Base, request.Revision, items), nil
}

func loadContract(ctx context.Context, location string) (normalize.Contract, error) {
	data, err := protocols.LoadSource(ctx, location)
	if err != nil {
		return normalize.Contract{}, err
	}
	parser := protoparser.NewParser(bytes.NewReader(data))
	definition, err := parser.Parse()
	if err != nil {
		return normalize.Contract{}, err
	}
	return normalizeDefinition(location, definition), nil
}

func normalizeDefinition(source string, definition *protoparser.Proto) normalize.Contract {
	contract := normalize.Contract{
		Protocol:  findings.ProtocolGRPC,
		Resources: []normalize.Resource{},
	}
	visitElements(source, "", "", definition.Elements, &contract.Resources)
	sort.Slice(contract.Resources, func(i, j int) bool {
		return contract.Resources[i].Identifier < contract.Resources[j].Identifier
	})
	return contract
}

func visitElements(source, protoPackage, parentMessage string, elements []protoparser.Visitee, target *[]normalize.Resource) {
	currentPackage := protoPackage
	for _, element := range elements {
		switch node := element.(type) {
		case *protoparser.Package:
			currentPackage = node.Name
		case *protoparser.Service:
			serviceName := qualify(currentPackage, node.Name)
			serviceID := resourceID("service", serviceName)
			*target = append(*target, normalize.Resource{
				Kind:       "service",
				Name:       serviceName,
				Identifier: serviceID,
				Source:     protocols.Source(source),
			})
			for _, child := range node.Elements {
				rpc, ok := child.(*protoparser.RPC)
				if !ok {
					continue
				}
				*target = append(*target, normalize.Resource{
					Kind:       "rpc",
					Name:       rpc.Name,
					Parent:     serviceID,
					Type:       fmt.Sprintf("%s->%s", rpc.RequestType, rpc.ReturnsType),
					Identifier: resourceID("rpc", serviceName, rpc.Name),
					Source:     protocols.Source(source),
				})
			}
		case *protoparser.Message:
			messageName := qualifyMessage(currentPackage, parentMessage, node.Name)
			visitMessage(source, currentPackage, messageName, node, target)
		}
	}
}

func visitMessage(source, protoPackage, messageName string, message *protoparser.Message, target *[]normalize.Resource) {
	for _, element := range message.Elements {
		switch node := element.(type) {
		case *protoparser.NormalField:
			label := "optional"
			if node.Required {
				label = "required"
			} else if node.Repeated {
				label = "repeated"
			}
			meta := map[string]string{
				"field_number": strconv.Itoa(node.Sequence),
				"label":        label,
			}
			*target = append(*target, normalize.Resource{
				Kind:       "message-field",
				Name:       node.Name,
				Parent:     messageName,
				Required:   node.Required,
				Type:       node.Type,
				Identifier: resourceID("message-field", messageName, node.Name),
				Meta:       meta,
				Source:     protocols.Source(source),
			})
		case *protoparser.Message:
			visitMessage(source, protoPackage, qualifyMessage(protoPackage, messageName, node.Name), node, target)
		}
	}
}

func buildFindings(base, revision normalize.Contract, changes diff.ContractDiff) []findings.Finding {
	var items []findings.Finding
	for _, removed := range changes.Removed {
		switch removed.Kind {
		case "service":
			items = append(items, protocols.Finding(findings.ProtocolGRPC, "protobuf.service.removed", findings.SeverityError, true, removed, "service removed", removed, nil))
		case "rpc":
			items = append(items, protocols.Finding(findings.ProtocolGRPC, "protobuf.rpc.removed", findings.SeverityError, true, removed, "rpc removed", removed, nil))
		case "message-field":
			items = append(items, protocols.Finding(findings.ProtocolGRPC, "protobuf.field.removed", findings.SeverityError, true, removed, "message field removed", removed, nil))
		}
	}
	for _, changed := range changes.Changed {
		before := *changed.Before
		after := *changed.After
		switch after.Kind {
		case "message-field":
			if before.Meta["field_number"] != "" && after.Meta["field_number"] != "" && before.Meta["field_number"] != after.Meta["field_number"] {
				items = append(items, protocols.ChangedFinding(findings.ProtocolGRPC, "protobuf.field.number_changed", findings.SeverityError, true, before, after, "field number changed"))
			}
			if !before.Required && after.Required {
				items = append(items, protocols.ChangedFinding(findings.ProtocolGRPC, "protobuf.field.required_tightened", findings.SeverityError, true, before, after, "field became required"))
			}
			if protocols.IncompatibleType(before.Type, after.Type) {
				items = append(items, protocols.ChangedFinding(findings.ProtocolGRPC, "protobuf.field.type_changed", findings.SeverityError, true, before, after, "field type changed incompatibly"))
			}
		case "rpc":
			if protocols.IncompatibleType(before.Type, after.Type) {
				items = append(items, protocols.ChangedFinding(findings.ProtocolGRPC, "protobuf.rpc.signature_changed", findings.SeverityError, true, before, after, "rpc request/response types changed"))
			}
		}
	}
	items = append(items, detectNumberReuse(base, revision)...)
	return items
}

func detectNumberReuse(base, revision normalize.Contract) []findings.Finding {
	baseNumbers := fieldNumbers(base)
	revNumbers := fieldNumbers(revision)
	var items []findings.Finding
	for key, beforeName := range baseNumbers {
		if afterName, ok := revNumbers[key]; ok && afterName != beforeName {
			resource := normalize.Resource{
				Kind:       "message-field",
				Name:       afterName,
				Identifier: resourceID("message-number", key),
				Meta:       map[string]string{"before_name": beforeName, "after_name": afterName},
			}
			items = append(items, protocols.Finding(findings.ProtocolGRPC, "protobuf.field_number.reused", findings.SeverityError, true, resource, "field number reused by a different field", beforeName, afterName))
		}
	}
	return items
}

func fieldNumbers(contract normalize.Contract) map[string]string {
	result := map[string]string{}
	for _, item := range contract.Resources {
		if item.Kind != "message-field" || item.Meta["field_number"] == "" {
			continue
		}
		key := item.Parent + "#" + item.Meta["field_number"]
		result[key] = item.Name
	}
	return result
}

func resourceID(kind string, parts ...string) string {
	return kind + "::" + strings.Join(parts, "::")
}

func qualify(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func qualifyMessage(protoPackage, parent, name string) string {
	if parent != "" {
		return parent + "." + name
	}
	return qualify(protoPackage, name)
}
