package diff

import "github.com/compatgate/compatgate/internal/normalize"

type ChangeKind string

const (
	ChangeAdded   ChangeKind = "added"
	ChangeRemoved ChangeKind = "removed"
	ChangeChanged ChangeKind = "changed"
)

type Change struct {
	Kind   ChangeKind
	Before *normalize.Resource
	After  *normalize.Resource
}

type ContractDiff struct {
	Added   []normalize.Resource
	Removed []normalize.Resource
	Changed []Change
}

func Compare(base normalize.Contract, revision normalize.Contract) ContractDiff {
	baseByID := make(map[string]normalize.Resource, len(base.Resources))
	revisionByID := make(map[string]normalize.Resource, len(revision.Resources))
	for _, resource := range base.Resources {
		baseByID[resource.Identifier] = resource
	}
	for _, resource := range revision.Resources {
		revisionByID[resource.Identifier] = resource
	}

	result := ContractDiff{}
	for id, before := range baseByID {
		after, ok := revisionByID[id]
		if !ok {
			result.Removed = append(result.Removed, before)
			continue
		}
		if resourceChanged(before, after) {
			beforeCopy := before
			afterCopy := after
			result.Changed = append(result.Changed, Change{Kind: ChangeChanged, Before: &beforeCopy, After: &afterCopy})
		}
	}

	for id, resource := range revisionByID {
		if _, ok := baseByID[id]; ok {
			continue
		}
		result.Added = append(result.Added, resource)
	}

	return result
}

func resourceChanged(before normalize.Resource, after normalize.Resource) bool {
	if before.Required != after.Required || before.Type != after.Type || before.Kind != after.Kind || before.Name != after.Name || before.Parent != after.Parent {
		return true
	}
	if len(before.EnumValues) != len(after.EnumValues) {
		return true
	}
	for index := range before.EnumValues {
		if before.EnumValues[index] != after.EnumValues[index] {
			return true
		}
	}
	if len(before.Meta) != len(after.Meta) {
		return true
	}
	for key, value := range before.Meta {
		if after.Meta[key] != value {
			return true
		}
	}
	return false
}
