package slackbot

import (
	"strings"
	"testing"
)

func TestTransitionTo(t *testing.T) {
	SetupTestDatabase()
	poll := &Poll{Stage: "stage1", UUID: "1"}
	poll.TransitionTo("stage2")
	if strings.Compare(poll.Stage, "stage2") != 0 {
		t.Fatal("expecting poll to be in stage: stage2, but got:", poll.Stage)
	}

	if strings.Compare(poll.PreviousStage, "stage1") != 0 {
		t.Fatal("expecting poll to be in stage: stage1, but got:", poll.Stage)
	}
}

func TestSlackPreviewAttachments(t *testing.T) {
	SetupTestDatabase()
	var testingTable = []struct {
		Input    *Poll
		Expected Attachment
	}{
		{
			// A perfect poll and attachment combo which expects all the fields
			// and no trouble
			Input: &Poll{
				Kind:     "response",
				UUID:     "1",
				Question: "The pig is in the pudding",
				Recipients: []Recipient{
					Recipient{SlackID: "derp", SlackName: "oh ya"},
					Recipient{SlackID: "derp2", SlackName: "oh ya"},
				},
				PossibleAnswers: []PossibleAnswer{
					PossibleAnswer{Value: "1"},
					PossibleAnswer{Value: "2"},
				},
			}, Expected: Attachment{
				Title:   "1",
				Pretext: "Look good to you (yes/no)?",
				Text:    "The pig is in the pudding",
				Fields: []AttachmentField{
					AttachmentField{
						Title: "Poll Type:",
						Value: "response",
						Short: true,
					},
					AttachmentField{
						Title: "# of Recipients:",
						Value: "2",
						Short: true,
					},
					AttachmentField{
						Title: "Possible Answers:",
						Value: "1, 2",
						Short: false,
					},
				},
			},
		},
	}

	for _, test := range testingTable {
		GetDB().Save(test.Input)
		result := test.Input.SlackPreviewAttachment()
		output := test.Expected
		if output.Title != result.Title {
			t.Fatal("Expected: ", output.Title, "got: ", result.Title)
		}

		if output.Pretext != result.Pretext {
			t.Fatal("Expected: ", output.Pretext, "got: ", result.Pretext)
		}

		if output.Text != result.Text {
			t.Error("Expected: ", output.Text, "got: ", result.Text)
		}

		if len(result.Fields) != len(output.Fields) {
			t.Error("Mismatched number of attachment fields. Expected: ", len(output.Fields), " but got: ", len(result.Fields))
		}

		for k, v := range result.Fields {
			expectedFieldValue := output.Fields[k].Value
			if expectedFieldValue != v.Value {
				t.Error("Expected: ", expectedFieldValue, " but got: ", v.Value)
			}
		}
	}
}

func TestAddAndGetSavedResponses(t *testing.T) {
	SetupTestDatabase()

	input := Poll{UUID: "responsetest"}
	if err := input.Save(); err != nil {
		t.Fatal("Did not expect there to be an issuing saving poll")
	}

	if err := input.AddResponse("bloop", "I am a valid response"); err != nil {
		t.Fatal("Expected no issue saving response")
	}
	output, _ := input.GetResponses()
	if len(output) != 1 {
		t.Fatal("Was expecting 1 response")
	}

	result := output[0]
	if strings.Compare(result.Value, "I am a valid response") != 0 {
		t.Fatal("Expecting response: I am a valid response but got:", result.Value)
	}
}

func TestResponseSummaryField(t *testing.T) {
	SetupTestDatabase()
	var testCases = []struct {
		Poll               Poll
		ExpectedAttachment string
	}{
		{
			Poll:               Poll{Kind: "response", Responses: []PollResponse{{Value: "Gorp"}}, Recipients: []Recipient{{SlackID: "derp"}}},
			ExpectedAttachment: "100% - 1 out of 1",
		},
	}
	for _, testCase := range testCases {
		GetDB().Save(&testCase.Poll)
		output := responseSummaryField(&testCase.Poll)
		if strings.Compare(output.Value, testCase.ExpectedAttachment) != 0 {
			t.Error("Expected attachment field: ", testCase.ExpectedAttachment, " but got: ", output.Value)
		}
	}

}
