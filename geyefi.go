package main

import (
	"archive/tar"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
  "log"
	"io"
	"io/ioutil"
  "net/http"
	"strconv"
	"strings"
)

const (
	UPLOAD_KEY = "818b6183a1a0839d88366f5d7a4b0161"
)

func credential(mac string, nonce string) (string, error) {
	// http://play.golang.org/p/rXnoiRvtzr
	h := md5.New()
	s := mac + nonce + UPLOAD_KEY
	b, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}
	h.Write(b)
	credential := fmt.Sprintf("%x", h.Sum(nil))
	return credential, nil
}

func respond(body []byte, resp http.ResponseWriter) {
	log.Printf("Responding %s\n", string(body))

	resp.Header().Set("Server", "Eye-Fi Agent/2.0.4.0 (Windows XP SP2)")
	resp.Header().Set("Pragma", "no-cache")
  resp.Header().Set("Content-Type", "text/xml; charset=\"utf-8\"") 
	resp.Header().Set("Content-Length", strconv.Itoa(len(body)))
	resp.Write(body)	

}

func handleStartSession(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	env, err := parseEnvelope(req.Body)
	if err != nil {
		log.Fatal(err)
	}

	credential, err := credential(env.Body.StartSession.MacAddress, env.Body.StartSession.Nonce)
	if err != nil {
		log.Fatal(err)
	}

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
		credential, "99208c155fc1883579cf0812ec0fe6d2", env.Body.StartSession.TransferMode, env.Body.StartSession.Timestamp, "false")
	responseBytes := []byte(response)


	// http://play.golang.org/p/Qtcle7j9EM
	/*
	var response Envelope
	response.Body.StartSessionResponse = &StartSessionResponse{}
	response.Body.StartSessionResponse.Credential = credential
	response.Body.StartSessionResponse.Nonce = "bbbbb"
	response.Body.StartSessionResponse.TransferMode = request.Body.StartSession.TransferMode
	response.Body.StartSessionResponse.Timestamp = request.Body.StartSession.Timestamp
	response.Body.StartSessionResponse.UpsyncAllowed = false

	responseBytes, err := xml.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}
	 */

	respond(responseBytes, resp)
}

func handleGetPhotoStatus(resp http.ResponseWriter, req *http.Request) {
//	defer req.Body.Close()
//	env, err := := parseEnvelope(req.Body)
//	if err != nil {
//		log.Fatal(err)
//	}

	// check card credential etc
	respond([]byte(
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>" +
		"<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\">" +
    "  <SOAP-ENV:Body>" +
		"    <ns1:GetPhotoStatusResponse xmlns:ns1=\"http://localhost/api/soap/eyefilm\">" +
    "      <fileid>1</fileid>" + 
    "      <offset>0</offset>" +
    "    </ns1:GetPhotoStatusResponse>" +
    "  </SOAP-ENV:Body>" +
		"</SOAP-ENV:Envelope>"), resp)
}

func handleUpload(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	mr, err := req.MultipartReader()
	if err != nil {
		log.Fatal(err)
	}

	soapPart, err := mr.NextPart()
	if err != nil {
		log.Fatal(err)
	}

	env, err := parseEnvelope(soapPart)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Getting file %s\n", env.Body.UploadPhoto.FileName)

	dataPart, err := mr.NextPart()
	if err != nil {
		log.Fatal(err)
	}

	tempDir, err := ioutil.TempDir("", "eyefi")
	if err != nil {
		log.Fatal(err)
	}

	tarReader := tar.NewReader(dataPart)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Decompressing: %s\n", header.Name)
		fname := fmt.Sprintf("%s/%s", tempDir, header.Name)

		data, err := ioutil.ReadAll(tarReader)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Writing %s\n", fname)
		ioutil.WriteFile(fname, data, 0777)
	}

	data, err := ioutil.ReadAll(dataPart)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Got %d bytes of data\n", len(data))

	checksumPart, err := mr.NextPart()
	if err != nil {
		log.Fatal(err)
	}

	checksum, err := ioutil.ReadAll(checksumPart)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Checksum %s\n", checksum)

	_, err = mr.NextPart()
	if err != io.EOF {
		log.Fatal("Got a fourth part!!");
	}
}

func parseEnvelope(in io.Reader) (*Envelope, error){
	body, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	log.Println(string(body))
	var envelope Envelope
	err = xml.Unmarshal(body, &envelope)
	if err != nil {
		return nil, err
	}

	return &envelope, nil
}

func handler(resp http.ResponseWriter, req *http.Request) {
  log.Printf(req.URL.Path)
	defer req.Body.Close()

	action := req.Header.Get("SoapAction")
	if strings.Contains(action, "urn:StartSession") {
		handleStartSession(resp, req)
	} else if strings.Contains(action, "urn:GetPhotoStatus") {
		handleGetPhotoStatus(resp, req)
	} else if (strings.Contains(req.URL.Path, "/api/soap/eyefilm/v1/upload")) {
		handleUpload(resp, req)
	} else {
		fmt.Printf("Unknown action:  %s\n", action)
	}
}

/*
2013/06/07 02:16:33 /api/soap/eyefilm/v1/upload
2013/06/07 02:16:34 -----------------------------02468ace13579bdfcafebabef00d
Content-Disposition: form-data; name="SOAPENVELOPE"

<?xml version="1.0" encoding="UTF-8"?><SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:ns1="EyeFi/SOAP/EyeFilm"><SOAP-ENV:Body><ns1:UploadPhoto><fileid>1</fileid><macaddress>0018562bbac0</macaddress><filename>IMG_0071.JPG.tar</filename><filesize>790016</filesize><filesignature>35310000f8f9030000000000e0110300</filesignature><encryption>none</encryption><flags>4</flags></ns1:UploadPhoto></SOAP-ENV:Body></SOAP-ENV:Envelope>
-----------------------------02468ace13579bdfcafebabef00d
Content-Disposition: form-data; name="FILENAME"; filename="IMG_0071.JPG.tar"
Content-Type: application/x-tar

<binary data>

-----------------------------02468ace13579bdfcafebabef00d
Content-Disposition: form-data; name="INTEGRITYDIGEST"

536b79c0509311aace14ae5981f76bc1
-----------------------------02468ace13579bdfcafebabef00d--


*/

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

type GetPhotoStatus struct {
	Credential string `xml:"credential"`
	MacAddress string `xml:"macaddress"`
	FileName string `xml:"filename"`
	Size int64 `xml:"filesize"`
	Signature string `xml:"filesignature"`
	Flags int64 `xml:"flags"`
}

type UploadPhoto struct {
	FileId int64 `xml:"fileid"`
	MacAddress string `xml:"macaddress"`
	FileName string `xml:"filename"`
	Size int64 `xml:"filesize"`
	Signature string `xml:"filesignature"`
	Flags int64 `xml:"flags"`	
}

type StartSessionResponse struct {
	Credential string `xml:"credential"`
	Nonce string `xml:"snonce"`
	TransferMode int32 `xml:transfermode"`
	Timestamp int32 `xml:"transfermodetimestamp"`
	UpsyncAllowed bool `xml:"upsyncallowed"`
}

type Body struct {
	StartSession *StartSession `xml:"EyeFi/SOAP/EyeFilm StartSession,omitempty"`
	StartSessionResponse *StartSessionResponse `xml:"EyeFi/SOAP/EyeFilm StartSessionResponse,omitempty"`
	UploadPhoto *UploadPhoto `xml:"EyeFi/SOAP/EyeFilm UploadPhoto,omitempty"`
}

type Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body Body `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
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
