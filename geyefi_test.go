package main

import (
	"encoding/xml"
  "testing"
)

func TestStartSessionRequestParse(t *testing.T) {
	xml := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:ns1=\"EyeFi/SOAP/EyeFilm\"><SOAP-ENV:Body><ns1:StartSession><macaddress>0018562bbac0</macaddress><cnonce>e8f2c769c23a2111d3e8aa07602e4814</cnonce><transfermode>2</transfermode><transfermodetimestamp>1364157918</transfermodetimestamp></ns1:StartSession></SOAP-ENV:Body></SOAP-ENV:Envelope>"
	expected := "e8f2c769c23a2111d3e8aa07602e4814"
	actual := parseNonce(xml)
	if actual != expected {
		t.Fatalf("Didn't get expected nonce: '%s' (Expected) vs '%s' (Actual)", expected, actual)
	}
}

func TestStartSessionResponseFormat(t *testing.T) {
	expected := "<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\"><SOAP-ENV:Body><ns1:StartSessionResponse xmlns:ns1=\"http://localhost/api/soap/eyefilm\"><credential>0d400f69ce6096c3771f9465c6123145</credential><snonce>d5b2b8dd7a681cfb5320aaac2fd9bba4</snonce><transfermode>2</transfermode><transfermodetimestamp>1304505230</transfermodetimestamp><upsyncallowed>false</upsyncallowed></ns1:StartSessionResponse></SOAP-ENV:Body></SOAP-ENV:Envelope>"

	var envelope Envelope
	envelope.NS = "http://schemas.xmlsoap.org/soap/envelope/"
	envelope.Body.StartSessionResponse = &StartSessionResponse{}
	envelope.Body.StartSessionResponse.Credential = "aaaaa"
	envelope.Body.StartSessionResponse.Nonce = "bbbbb"
	envelope.Body.StartSessionResponse.TransferMode = 1
	envelope.Body.StartSessionResponse.Timestamp = 2
	envelope.Body.StartSessionResponse.UpsyncAllowed = false

	bytes, err := xml.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(bytes)

	if actual != expected {
		t.Fatalf("Didn't get expected nonce: '%s' (Expected) vs '%s' (Actual)", expected, actual)
	}
}
