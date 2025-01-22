package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
)

// Sends transaction result data to Telegram Group Chat after the transaction has been confirmed
func (ns *NSReceiver) send_telegram_updates(body_received string) {
	var response *http.Response

	apis_obj, err := get_apis_obj()
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		os.Exit(1)
	}

	url := apis_obj["url"].(map[string]interface{})["telegram"].(string)

	obj := make(map[string]interface{})
	obj["chat_id"] = apis_obj["chat_id"]
	obj["text"] = body_received
	body, _ := json.Marshal(obj)
	response, err = http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		ns.Log.Error().Msg(err.Error())
		return
	}
	defer response.Body.Close()
}
