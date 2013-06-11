package geyefi

/*
 * TODO
 * - Verify credentials from card
 * - Verify checksum
 * - Better tests
 * - Fix response (either figure out how to use xml library, or use templates, not strings)
 */
import (
	"archive/tar"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"fmt"
  "log"
	"io"
	"io/ioutil"
	"net"
  "net/http"
	"strconv"
	"strings"
)

////// API ///////

type UploadHandler interface {
	// should data be a Reader?
	HandleUpload(filename string, data []byte) error
}

func NewServer(uploadKey string, uploadHandler UploadHandler) *Server {
	return &Server{uploadKey: uploadKey, uploadHandler: uploadHandler, port: 59278}
}


func (e *Server) ListenAndServe() {
	log.Println("Serving")

	http.HandleFunc("/", e.handler)
	http.ListenAndServe(fmt.Sprintf(":%d", e.port), nil)
}


/*
func (e* Server) ListenAndServe() error {
	// TODO: mutex
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", e.port))
	if err != nil {
		return err
	}

	e.listener = &l
	e.server = &http.Server{}
	return e.server.Serve(l)
}

func (e *Server) Stop() {
	// TODO: mutex
	if e.listener != nil {
		(*e.listener).Close()
	}
	e.listener = nil
	e.server = nil
}
*/
/////////////

type SaveFileHandler struct {
	Directory string
}

func (h *SaveFileHandler) HandleUpload(filename string, data []byte) error {
	localFilename := fmt.Sprintf("%s/%s", h.Directory, filename)
	log.Printf("Writing %s\n", localFilename)
	return ioutil.WriteFile(localFilename, data, 0777)
}


/* Example

func main() {
	tempDir, err := ioutil.TempDir("", "eyefi")
	if err != nil {
		log.Fatal(err)
	}

	handler := &SaveFileHandler{Directory: tempDir}
	log.Printf("Files will be saved to: %s\n", tempDir)
	e := NewServer("818b6183a1a0839d88366f5d7a4b0161", handler)
	e.ListenAndServe()
}

*/

//////////////////

type Server struct {
	uploadKey string
	uploadHandler UploadHandler
	port int

	listener *net.Listener
	server *http.Server
}

func (e *Server) handler(resp http.ResponseWriter, req *http.Request) {
	log.Printf(req.URL.Path)
	defer req.Body.Close()

	action := req.Header.Get("SoapAction")
	var err error
	if strings.Contains(action, "urn:StartSession") {
		err = e.handleStartSession(resp, req)
	} else if strings.Contains(action, "urn:GetPhotoStatus") {
		err = handleGetPhotoStatus(resp, req)
	} else if (strings.Contains(req.URL.Path, "/api/soap/eyefilm/v1/upload")) {
		err = e.handleUpload(resp, req)
	} else if strings.Contains(action, "urn:MarkLastPhotoInRoll") {
		err = handleMarkLastPhotoInRoll(resp, req)
	} else {
		fmt.Printf("Unknown action:  %s\n", action)
	}

	if err != nil {
		log.Printf("ERROR HANDLING REQUEST: %s\n", err)
	}
}

func (e *Server) credential(mac string, nonce string) (string, error) {
	h := md5.New()
	s := mac + nonce + e.uploadKey
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

func (e *Server) handleStartSession(resp http.ResponseWriter, req *http.Request) error {
	defer req.Body.Close()
	env, err := parseEnvelope(req.Body)
	if err != nil {
		return err
	}

	credential, err := e.credential(env.Body.StartSession.MacAddress, env.Body.StartSession.Nonce)
	if err != nil {
		return err
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
	return nil
}

func handleGetPhotoStatus(resp http.ResponseWriter, req *http.Request) error {
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

	return nil
}

func (e *Server) consumeUpload(req *http.Request) error {
	mr, err := req.MultipartReader()
	if err != nil {
		return err
	}

	soapPart, err := mr.NextPart()
	if err != nil {
		return err
	}

	env, err := parseEnvelope(soapPart)
	if err != nil {
		return err
	}
	log.Printf("Getting file %s\n", env.Body.UploadPhoto.FileName)

	dataPart, err := mr.NextPart()
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(dataPart)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fmt.Printf("Decompressing: %s\n", header.Name)

		data, err := ioutil.ReadAll(tarReader)
		if err != nil {
			return err
		}

		err = e.uploadHandler.HandleUpload(header.Name, data)
		if err != nil {
			return err
		}
	}

	// TODO: verify checksum
	checksumPart, err := mr.NextPart()
	if err != nil {
		return err
	}

	checksum, err := ioutil.ReadAll(checksumPart)
	if err != nil {
		return err
	}
	log.Printf("Checksum %s\n", checksum)

	_, err = mr.NextPart()
	if err != io.EOF {
		return err
	}
	return nil
}

func (e *Server) handleUpload(resp http.ResponseWriter, req *http.Request) error {
	defer req.Body.Close()

	err := e.consumeUpload(req)

	success := "true"
	if err != nil {
		success = "false"
		log.Printf("ERROR FROM UPLOAD HANDLER: %s\n", err)
	}

	response := fmt.Sprintf(
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>" +
		"<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\">" +
    "  <SOAP-ENV:Body>" +
    "    <ns1:UploadPhotoResponse xmlns:ns1=\"http://localhost/api/soap/eyefilm\">" +
    "      <success>%s</success>" +
    "    </ns1:UploadPhotoResponse>" +
		"  </SOAP-ENV:Body>" +
		"</SOAP-ENV:Envelope>", success)

	respond([]byte(response), resp)
	return nil
}

func handleMarkLastPhotoInRoll(resp http.ResponseWriter, req *http.Request) error {
//	defer req.Body.Close()
//	env, err := parseEnvelope(req.Body)
//	if err != nil {
//		log.Fatal(err)
//	}

	response := fmt.Sprintf(
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>" +
		"<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\">" +
    "  <SOAP-ENV:Body>" +
		"    <ns1:MarkLastPhotoInRollResponse xmlns:ns1=\"http://localhost/api/soap/eyefilm\" />" +
		"  </SOAP-ENV:Body>" +
		"</SOAP-ENV:Envelope>")

	respond([]byte(response), resp)
	return nil
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

type MarkLastPhotoInRoll struct {
	MacAddress string `xml:"macaddress"`
	MergeDelta string `xml:"mergedelta"`
}

//type StartSessionResponse struct {
//	Credential string `xml:"credential"`
//	Nonce string `xml:"snonce"`
//	TransferMode int32 `xml:transfermode"`
//	Timestamp int32 `xml:"transfermodetimestamp"`
//	UpsyncAllowed bool `xml:"upsyncallowed"`
//}

type Body struct {
	StartSession *StartSession `xml:"EyeFi/SOAP/EyeFilm StartSession,omitempty"`
//	StartSessionResponse *StartSessionResponse `xml:"EyeFi/SOAP/EyeFilm StartSessionResponse,omitempty"`
	UploadPhoto *UploadPhoto `xml:"EyeFi/SOAP/EyeFilm UploadPhoto,omitempty"`
	MarkLastPhotoInRoll *MarkLastPhotoInRoll `xml:"EyeFi/SOAP/EyeFilm MarkLastPhotoInRoll,omitempty"`
}

type Envelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body Body `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}


