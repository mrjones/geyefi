package geyefi

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
  "testing"
)

const (
	UPLOAD_KEY = "abcd"
)

func TestSession(t *testing.T) {
	server := NewServer(UPLOAD_KEY, &SaveFileHandler{Directory: "/tmp"})

	port := 12121 // pick something at random
	server.port = port

	go server.ListenAndServe()


	// Start a session
	startUrl := fmt.Sprintf("http://localhost:%d/api/soap/eyefilm/v1", port)
	startBody := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:ns1=\"EyeFi/SOAP/EyeFilm\"><SOAP-ENV:Body><ns1:StartSession><macaddress>0018562bbac0</macaddress><cnonce>e8f2c769c23a2111d3e8aa07602e4814</cnonce><transfermode>2</transfermode><transfermodetimestamp>1364157918</transfermodetimestamp></ns1:StartSession></SOAP-ENV:Body></SOAP-ENV:Envelope>"
	startRequest, err := http.NewRequest("POST", startUrl, strings.NewReader(startBody))
	if err != nil {
		t.Fatal(err)
	}
	startRequest.Header.Set("SoapAction", "urn:StartSession")

	client := http.Client{}
	resp, err := client.Do(startRequest)

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var envelope Envelope
	err = xml.Unmarshal(body, &envelope)
	if err != nil {
		t.Fatal(err)
	}

	// python -c  "mac=\"0018562bbac0\"; nonce=\"e8f2c769c23a2111d3e8aa07602e4814\"; key=\"abcd\"; import hashlib; import binascii; m = hashlib.md5(); m.update(binascii.unhexlify(mac + nonce + key)); print m.hexdigest()" 
	expectedCredential := "f561e60acb9145efe363ecd3efdd8588"
	if envelope.Body.StartSessionResponse == nil {
		t.Fatalf("Error parsing: '%s'\n", string(body))
	}
	if envelope.Body.StartSessionResponse.Credential != expectedCredential {
		t.Fatalf("Expected credential '%s' does not match actual credential '%s'\n",
			expectedCredential, envelope.Body.StartSessionResponse.Credential)
	}

	

//	server.Stop()
}


/*
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
*/
