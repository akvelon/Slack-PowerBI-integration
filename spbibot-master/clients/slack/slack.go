package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"


)

// RespondNow is for immediate responses to a slash command.
func RespondNow(w http.ResponseWriter, m *slack.Msg) (err error) {
	j, err := json.Marshal(m)
	if err != nil {
		return err
	}

	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeJSON)
	_, err = w.Write(j)

	return
}

// RespondLater is for delayed responses to a slash command.
func RespondLater(s *slack.SlashCommand, m *slack.Msg) error {
	j, err := json.Marshal(m)
	if err != nil {
		return err
	}

	headers := map[string]string{
		constants.HTTPHeaderContentType: constants.MIMETypeJSON,
	}

	res, err := client.HandleHTTPRequest(http.MethodPost, s.ResponseURL, headers, bytes.NewReader(j), true)
	if err != nil {
		return err
	}

	if s := res.StatusCode; s != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", s)
	}

	return nil
}

// SendValidationError sends an input block validation error message
func SendValidationError(w http.ResponseWriter, inputBlockID, msg string) error {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeJSON)
	w.WriteHeader(http.StatusOK)

	r := slack.NewErrorsViewSubmissionResponse(map[string]string{inputBlockID: msg})
	j, err := json.Marshal(r)
	if err != nil {
		return err
	}

	_, err = w.Write(j)

	return err
}

// IsInConversation checks bot's conversation membership.
func IsInConversation(api *slack.Client, conversationID string) (bool, error) {
	c, err := api.GetConversationInfo(conversationID, false)
	if err != nil {
		return false, err
	}

	return (c.IsChannel || c.IsGroup || c.IsMpIM) && c.IsMember || c.IsIM, nil
}

// UpdateView updates a view
func UpdateView(w http.ResponseWriter, v *slack.ModalViewRequest) error {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeJSON)
	w.WriteHeader(http.StatusOK)

	p := slack.NewUpdateViewSubmissionResponse(v)
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = w.Write(j)

	return err
}

// PushView adds a new view to a view stack.
func PushView(w http.ResponseWriter, v *slack.ModalViewRequest) error {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeJSON)
	w.WriteHeader(http.StatusOK)

	p := slack.NewPushViewSubmissionResponse(v)
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = w.Write(j)

	return err
}

// ClearView clears a view stack.
func ClearView(w http.ResponseWriter) error {
	w.Header().Set(constants.HTTPHeaderContentType, constants.MIMETypeJSON)
	w.WriteHeader(http.StatusOK)

	p := slack.NewClearViewSubmissionResponse()
	j, err := json.Marshal(p)
	if err != nil {
		return err
	}

	_, err = w.Write(j)

	return err
}

// Ack performs an acknowledgement response.
func Ack(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte{})

	return err
}
