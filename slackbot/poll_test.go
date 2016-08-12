package slackbot

import (
	"strings"
	"testing"
)

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
						Title: "Recipients:",
						Value: "<@derp>, <@derp2>",
						Short: false,
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
			t.Error("Expected: ", output.Title, "got: ", result.Title)
		}

		if output.Pretext != result.Pretext {
			t.Error("Expected: ", output.Pretext, "got: ", result.Pretext)
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
