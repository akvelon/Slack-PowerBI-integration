package teams

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"



	"github.com/avast/retry-go"
)

const (
	graphEndpoint = "https://graph.microsoft.com/v1.0/"
	methodPOST    = "POST"
	retryAttempts = 5
	failedMessage = "{\"body\": {\"content\": \"Sorry, we couldn't generate report %v\"}}"
)

func sendRequest(method string, u string, headers map[string]string, body string) error {
	payload := strings.NewReader(body)

	client := &http.Client{}
	address, _ := url.Parse(graphEndpoint)
	address.Path = path.Join(address.Path, u)
	u = address.String()
	req, err := http.NewRequest(method, u, payload)

	if err != nil {
		return err
	}

	for h, v := range headers {
		req.Header.Add(h, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	// Checked, if status code isn't 2**, then exception
	if res.StatusCode/100 != 2 {
		return domain.ErrUnexpectedStatusCode(res.StatusCode)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(res.Body)

	return nil
}

func getHeadersAndEndpoint(authToken string, o *utils.ShareOptions) (map[string]string, string) {
	headers := make(map[string]string)

	headers["Content-Type"] = "application/json"
	headers["Authorization"] = fmt.Sprintf("Bearer %v", authToken)

	endpoint := fmt.Sprintf("/teams/%v/channels/%v/messages", o.WorkspaceID, o.ChannelID)

	return headers, endpoint
}

func SendMessage(encodedReport string, o *utils.ShareOptions, botToken string) error {
	err := retry.Do(
		func() error {
			body := fmt.Sprintf("{\n  \"messageType\": \"message\",\n  \"body\": {\n    \"contentType\": \"html\",\n    \"content\": \"Report generated</br><div><span><img height=\\\"720\\\" src=\\\"../hostedContents/1/$value\\\" width=\\\"1280\\\" style=\\\"vertical-align:bottom; width:1280px; height:720px\\\"></span>\\n</div>\"\n  },\n  \"hostedContents\": [\n    {\n      \"@microsoft.graph.temporaryId\": \"1\",\n      \"contentBytes\": \"%s\",\n      \"contentType\": \"image/png\"\n    }\n  ]\n}", encodedReport)
			headers, endpoint := getHeadersAndEndpoint(botToken, o)

			err := sendRequest(methodPOST, endpoint, headers, body)

			return err
		},
		retry.Attempts(retryAttempts),
	)

	return err
}

func SendFailedMessage(o *utils.ShareOptions, botToken string) error {
	err := retry.Do(
		func() error {
			body := fmt.Sprintf(failedMessage, o.ReportName)
			headers, endpoint := getHeadersAndEndpoint(botToken, o)

			err := sendRequest(methodPOST, endpoint, headers, body)

			return err
		},
		retry.Attempts(retryAttempts),
	)

	return err
}
