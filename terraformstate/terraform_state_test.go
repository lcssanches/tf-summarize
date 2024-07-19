package terraformstate

import (
	"testing"

	. "github.com/hashicorp/terraform-json"
	tfjson "github.com/hashicorp/terraform-json"

	"github.com/stretchr/testify/assert"
)

func TestResourceChangeColor(t *testing.T) {
	ExpectedColors := map[Action]string{
		ActionCreate: ColorGreen,
		ActionDelete: ColorRed,
		ActionUpdate: ColorYellow,
	}

	for action, expectedColor := range ExpectedColors {
		create := &ResourceChange{Change: &Change{Actions: []Action{action}}}
		color, _ := GetColorPrefixAndSuffixText(create)

		assert.Equal(t, color, expectedColor)
	}

	CreateDelete := &ResourceChange{Change: &Change{Actions: []Action{ActionCreate, ActionDelete}}}
	color, _ := GetColorPrefixAndSuffixText(CreateDelete)
	assert.Equal(t, color, ColorMagenta)

	DeleteCreate := &ResourceChange{Change: &Change{Actions: []Action{ActionDelete, ActionCreate}}}
	color, _ = GetColorPrefixAndSuffixText(DeleteCreate)
	assert.Equal(t, color, ColorMagenta)
}

func TestGetAllResourceChanges(t *testing.T) {
	input := []byte(`
	{
		"resource_changes": [
			{
				"address": "aws_instance.example1",
				"change": {
					"actions": ["create"]
				}
			},
			{
				"address": "aws_instance.example2",
				"change": {
					"actions": ["delete"]
				}
			},
			{
				"address": "aws_instance.example3",
				"change": {
					"actions": ["update"]
				}
			},
			{
				"address": "aws_instance.example4",
				"change": {
					"actions": ["create", "delete"]
				}
			},
			{
				"address": "aws_instance.example5",
				"change": {
					"importing": {
						"id": "example5"
					}
				}
			}
		]
	}`)

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("failed to parse input: %v", err)
	}

	expected := map[string]ResourceChanges{
		"import": {
			&tfjson.ResourceChange{
				Address: "aws_instance.example5",
				Change: tfjson.Change{
					Importing: &tfjson.Import{
						ID: "example5",
					},
				},
			},
		},
		"add": {
			&tfjson.ResourceChange{
				Address: "aws_instance.example1",
				Change: tfjson.Change{
					Actions: []string{"create"},
				},
			},
		},
		"delete": {
			&tfjson.ResourceChange{
				Address: "aws_instance.example2",
				Change: tfjson.Change{
					Actions: []string{"delete"},
				},
			},
		},
		"update": {
			&tfjson.ResourceChange{
				Address: "aws_instance.example3",
				Change: tfjson.Change{
					Actions: []string{"update"},
				},
			},
		},
		"recreate": {
			&tfjson.ResourceChange{
				Address: "aws_instance.example4",
				Change: tfjson.Change{
					Actions: []string{"create", "delete"},
				},
			},
		},
	}

	result := GetAllResourceChanges(plan)

	for key, expectedChanges := range expected {
		if len(result[key]) != len(expectedChanges) {
			t.Errorf("Expected length of %s to be %d, got %d", key, len(expectedChanges), len(result[key]))
		}
		for i, expectedChange := range expectedChanges {
			if result[key][i].Address != expectedChange.Address {
				t.Errorf("Expected %s address at index %d to be %s, got %s", key, i, expectedChange.Address, result[key][i].Address)
			}
		}
	}
}

func TestGetAllOutputChanges(t *testing.T) {
	input := []byte(`
	{
		"output_changes": {
			"output1": {
				"actions": ["create"]
			},
			"output2": {
				"actions": ["delete"]
			},
			"output3": {
				"actions": ["update"]
			}
		}
	}`)

	plan := tfjson.Plan{}
	err := json.Unmarshal(input, &plan)
	if err != nil {
		t.Fatalf("failed to parse input: %v", err)
	}

	expected := map[string][]string{
		"add":    {"output1"},
		"delete": {"output2"},
		"update": {"output3"},
	}

	result := GetAllOutputChanges(plan)

	for key, expectedOutputs := range expected {
		if len(result[key]) != len(expectedOutputs)) {
			t.Errorf("Expected length of %s to be %d, got %d", key, len(expectedOutputs), len(result[key]))
		}
		for i, expectedOutput := range expectedOutputs {
			if result[key][i] != expectedOutput {
				t.Errorf("Expected %s output at index %d to be %s, got %s", key, i, expectedOutput, result[key][i])
			}
		}
	}
}

func TestResourceChangeSuffix(t *testing.T) {
	ExpectedSuffix := map[Action]string{
		ActionCreate: "(+)",
		ActionDelete: "(-)",
		ActionUpdate: "(~)",
	}

	for action, expectedSuffix := range ExpectedSuffix {
		create := &ResourceChange{Change: &Change{Actions: []Action{action}}}
		_, suffix := GetColorPrefixAndSuffixText(create)

		assert.Equal(t, suffix, expectedSuffix)
	}
	CreateDelete := &ResourceChange{Change: &Change{Actions: []Action{ActionCreate, ActionDelete}}}
	_, suffix := GetColorPrefixAndSuffixText(CreateDelete)
	assert.Equal(t, suffix, "(+/-)")

	DeleteCreate := &ResourceChange{Change: &Change{Actions: []Action{ActionDelete, ActionCreate}}}
	_, suffix = GetColorPrefixAndSuffixText(DeleteCreate)
	assert.Equal(t, suffix, "(-/+)")
}

func TestResourceChangeColorAndSuffixImport(t *testing.T) {
	importing := &ResourceChange{Change: &Change{Importing: &Importing{ID: "id"}}}
	color, suffix := GetColorPrefixAndSuffixText(importing)

	assert.Equal(t, color, ColorCyan)
	assert.Equal(t, suffix, "(i)")
}

func TestFilterNoOpResources(t *testing.T) {
	resourceChanges := ResourceChanges{
		&ResourceChange{Address: "no-op1", Change: &Change{Actions: Actions{ActionNoop}}},
		&ResourceChange{Address: "no-op3", Change: &Change{Actions: Actions{ActionNoop}, Importing: nil}},
		&ResourceChange{Address: "no-op2", Change: &Change{Actions: Actions{ActionNoop}, Importing: &Importing{ID: ""}}},
		&ResourceChange{Address: "create", Change: &Change{Actions: Actions{ActionCreate}}},
	}
	plan := tfjson.Plan{ResourceChanges: resourceChanges}

	FilterNoOpResources(&plan)

	expectedResourceChangesAfterFiltering := ResourceChanges{
		&ResourceChange{Address: "create", Change: &Change{Actions: Actions{ActionCreate}}},
	}
	assert.Equal(t, expectedResourceChangesAfterFiltering, plan.ResourceChanges)
}
