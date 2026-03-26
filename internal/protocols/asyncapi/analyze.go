package asyncapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/compatgate/compatgate/internal/diff"
	"github.com/compatgate/compatgate/internal/findings"
	"github.com/compatgate/compatgate/internal/normalize"
	"github.com/compatgate/compatgate/internal/protocols"
	"github.com/compatgate/compatgate/pkg/compatgate"
	"gopkg.in/yaml.v3"
)

type document struct {
	Channels map[string]channel `yaml:"channels"`
}

type channel struct {
	Publish   *operation `yaml:"publish"`
	Subscribe *operation `yaml:"subscribe"`
}

type operation struct {
	Message *message `yaml:"message"`
}

type message struct {
	Payload *schema `yaml:"payload"`
}

type schema struct {
	Type       string             `yaml:"type"`
	Required   []string           `yaml:"required"`
	Properties map[string]*schema `yaml:"properties"`
	Items      *schema            `yaml:"items"`
}

func Analyze(ctx context.Context, request compatgate.Request) (findings.Report, error) {
	baseDoc, err := parse(ctx, request.Base)
	if err != nil {
		return findings.Report{}, err
	}
	revisionDoc, err := parse(ctx, request.Revision)
	if err != nil {
		return findings.Report{}, err
	}
	result := diff.Compare(normalizeDoc(baseDoc, request.Base), normalizeDoc(revisionDoc, request.Revision))
	items := evaluate(result)
	sort.Slice(items, func(i, j int) bool { return items[i].RuleID < items[j].RuleID })
	return findings.NewReport([]findings.Protocol{findings.ProtocolAsyncAPI}, request.Base, request.Revision, items), nil
}

func parse(ctx context.Context, location string) (document, error) {
	data, err := protocols.LoadSource(ctx, location)
	if err != nil {
		return document{}, err
	}
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return document{}, err
	}
	return doc, nil
}

func normalizeDoc(doc document, file string) normalize.Contract {
	contract := normalize.Contract{Protocol: findings.ProtocolAsyncAPI}
	for channelName, entry := range doc.Channels {
		channelID := "channel:" + channelName
		contract.Resources = append(contract.Resources, normalize.Resource{
			Kind:       "channel",
			Name:       channelName,
			Identifier: channelID,
			Source:     &findings.SourceLocation{File: file},
		})
		for _, item := range []struct {
			name string
			op   *operation
		}{
			{name: "publish", op: entry.Publish},
			{name: "subscribe", op: entry.Subscribe},
		} {
			if item.op == nil {
				continue
			}
			opID := fmt.Sprintf("operation:%s:%s", channelName, item.name)
			contract.Resources = append(contract.Resources, normalize.Resource{
				Kind:       "operation",
				Name:       item.name,
				Parent:     channelName,
				Identifier: opID,
				Source:     &findings.SourceLocation{File: file},
			})
			if item.op.Message != nil && item.op.Message.Payload != nil {
				contract.Resources = append(contract.Resources, walkPayload(opID, item.op.Message.Payload, file)...)
			}
		}
	}
	return contract
}

func walkPayload(parent string, payload *schema, file string) []normalize.Resource {
	if payload == nil {
		return nil
	}
	requiredSet := make(map[string]bool, len(payload.Required))
	for _, name := range payload.Required {
		requiredSet[name] = true
	}
	keys := make([]string, 0, len(payload.Properties))
	for key := range payload.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]normalize.Resource, 0)
	for _, name := range keys {
		child := payload.Properties[name]
		id := parent + ":" + name
		resource := normalize.Resource{
			Kind:       "message-field",
			Name:       name,
			Parent:     parent,
			Identifier: id,
			Required:   requiredSet[name],
			Type:       child.Type,
			Source:     &findings.SourceLocation{File: file},
		}
		items = append(items, resource)
		items = append(items, walkPayload(id, child, file)...)
	}
	if payload.Items != nil {
		items = append(items, walkPayload(parent+"[]", payload.Items, file)...)
	}
	return items
}

func evaluate(result diff.ContractDiff) []findings.Finding {
	items := make([]findings.Finding, 0)
	for _, removed := range result.Removed {
		switch removed.Kind {
		case "channel":
			items = append(items, finding("asyncapi.channel.removed", removed, "channel removed"))
		case "operation":
			items = append(items, finding("asyncapi.operation.removed", removed, "operation removed"))
		case "message-field":
			items = append(items, finding("asyncapi.payload_field.removed", removed, "payload field removed"))
		}
	}
	for _, added := range result.Added {
		if added.Kind == "message-field" && added.Required {
			items = append(items, finding("asyncapi.payload_field.required_added", added, "required payload field added"))
		}
	}
	for _, changed := range result.Changed {
		before, after := *changed.Before, *changed.After
		if after.Kind != "message-field" {
			continue
		}
		if !before.Required && after.Required {
			items = append(items, changedFinding("asyncapi.payload_field.required_tightened", before, after, "payload field became required"))
		}
		if before.Type != after.Type && before.Type != "" && after.Type != "" {
			items = append(items, changedFinding("asyncapi.payload_field.type_changed", before, after, "payload field type changed"))
		}
	}
	return items
}

func finding(ruleID string, resource normalize.Resource, message string) findings.Finding {
	return protocols.Finding(findings.ProtocolAsyncAPI, ruleID, findings.SeverityError, true, resource, message, nil, nil)
}

func changedFinding(ruleID string, before normalize.Resource, after normalize.Resource, message string) findings.Finding {
	return protocols.ChangedFinding(findings.ProtocolAsyncAPI, ruleID, findings.SeverityError, true, before, after, message)
}
