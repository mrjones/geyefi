package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
  "log"
	"io/ioutil"
  "net/http"
	"strconv"
)

const (
	UPLOAD_KEY = "818b6183a1a0839d88366f5d7a4b0161"
)

func handler(resp http.ResponseWriter, req *http.Request) {
  log.Printf(req.URL.Path)
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(string(body))
	var request Envelope
	err = xml.Unmarshal(body, &request)
	if err != nil {
		log.Fatal(err)
	}

	// http://play.golang.org/p/rXnoiRvtzr
	h := md5.New()
//	s := request.Body.StartSession.MacAddress + UPLOAD_KEY + request.Body.StartSession.Nonce
	s := request.Body.StartSession.MacAddress + request.Body.StartSession.Nonce + UPLOAD_KEY
	log.Printf("s = %s %s %s -> %s\n", request.Body.StartSession.MacAddress, UPLOAD_KEY, request.Body.StartSession.Nonce, s)
	b, err := hex.DecodeString(s)
	if err != nil {
		log.Fatal(err)
	}
	h.Write(b)
	credential := fmt.Sprintf("%x", h.Sum(nil))

	response := fmt.Sprintf(
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>" +
		"<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\">" +
    "  <SOAP-ENV:Body>" +
    "    <StartSessionResponse xmlns=\"http://localhost/api/soap/eyefilm\">" +
    "      <credential>%s</credential>" +
    "      <snonce>%s</snonce>" +
    "      <transfermode>%d</transfermode>" +
    "      <transfermodetimestamp>%d</transfermodetimestamp>" +
    "      <upsyncallowed>%s</upsyncallowed>" +
    "    </StartSessionResponse>" +
    "  </SOAP-ENV:Body>" +
		"</SOAP-ENV:Envelope>",
		credential, "99208c155fc1883579cf0812ec0fe6d2", request.Body.StartSession.TransferMode, request.Body.StartSession.Timestamp, "false")

//	response := fmt.Sprintf(
//		"<?xml version=\"1.0\" encoding=\"UTF-8\"?><SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\"><SOAP-ENV:Body><ns1:StartSessionResponse xmlns:ns1=\"http://localhost/api/soap/eyefilm\"><credential>%s</credential><snonce>%s</snonce><transfermode>%d</transfermode><transfermodetimestamp>%d</transfermodetimestamp><upsyncallowed>%s</upsyncallowed></ns1:StartSessionResponse></SOAP-ENV:Body></SOAP-ENV:Envelope>",
//	credential, "99208c155fc1883579cf0812ec0fe6d2", request.Body.StartSession.TransferMode, request.Body.StartSession.Timestamp, "false")


//	var response Envelope
//	response.Body.StartSessionResponse = &StartSessionResponse{}
//	response.Body.StartSessionResponse.Credential = credential
//	response.Body.StartSessionResponse.Nonce = "bbbbb"
//	response.Body.StartSessionResponse.TransferMode = request.Body.StartSession.TransferMode
//	response.Body.StartSessionResponse.Timestamp = request.Body.StartSession.Timestamp
//	response.Body.StartSessionResponse.UpsyncAllowed = false

//	responseBytes, err := xml.Marshal(response)
//	if err != nil {
//		log.Fatal(err)
//	}

	log.Printf("Responding: %s\n", response)

	resp.Header().Set("Server", "Eye-Fi Agent/2.0.4.0 (Windows XP SP2)")
	resp.Header().Set("Pragma", "no-cache")
  resp.Header().Set("Content-Type", "text/xml; charset=\"utf-8\"") 
	resp.Header().Set("Content-Length", strconv.Itoa(len([]byte(response))))
	resp.Write([]byte(response))
}

func main() {
  log.Println("Hello, world!")

  http.HandleFunc("/", handler)
  http.ListenAndServe(":59278", nil)
}

type StartSession struct {
	MacAddress string `xml:"macaddress"`
	Nonce string `xml:"cnonce"`
	TransferMode int32 `xml:"transfermode"`
	Timestamp int32 `xml:"transfermodetimestamp"`
}

type StartSessionResponse struct {
	Credential string `xml:"credential"`
	Nonce string `xml:"snonce"`
	TransferMode int32 `xml:transfermode"`
	Timestamp int32 `xml:"transfermodetimestamp"`
	UpsyncAllowed bool `xml:"upsyncallowed"`
}

type Body struct {
	StartSession *StartSession `xml:"StartSession,omitempty"`
//	StartSessionResponse *StartSessionResponse `xml:"EyeFi/SOAP/EyeFilm StartSessionResponse,omitempty"`
}

type Envelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body Body `xml:"Body"`
}

// return err
func parseNonce(startSessionXml string) string {
	var envelope Envelope
	err := xml.Unmarshal([]byte(startSessionXml), &envelope)
	if err != nil {
		log.Fatal(err)
	}
	return envelope.Body.StartSession.Nonce
}
