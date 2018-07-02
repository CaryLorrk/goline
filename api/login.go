package api

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"git.apache.org/thrift.git/lib/go/thrift"
	prot "github.com/carylorrk/goline/protocol"
)

func getEmailRegexFactory() func() *regexp.Regexp {
	emailRegex, _ := regexp.Compile("[^@]+@[^@]+\\.[^@]")
	return func() *regexp.Regexp {
		return emailRegex
	}

}

var emailRegex *regexp.Regexp = getEmailRegexFactory()()

func lookupIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func lookupHostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "My Computer"
	}
	return name
}

func (self *LineClient) AuthTokenLogin(authToken string) error {
	httpTrans := self.client.Transport.(*thrift.THttpClient)
	self.header.Add("X-Line-Access", authToken)
	httpTrans.SetHeader("X-Line-Access", authToken)
	self.AuthToken = authToken

	_, err := self.RefreshRevision()
	if err != nil {
		return err
	}

	_, err = self.RefreshProfile()
	if err != nil {
		return err
	}
	_, err = self.RefreshGroups()
	if err != nil {
		return err
	}
	_, err = self.RefreshContacts()
	if err != nil {
		return err
	}

	_, err = self.RefreshRooms()
	if err != nil {
		return err
	}
	return nil
}

func getJson(url string, header *http.Header) (map[string]interface{}, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = *header
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(res.Body)
	if err != nil {
		return nil, err
	}
	var jsonMap map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &jsonMap)
	return jsonMap, err
}

func (self *LineClient) GetPincode(id string, password string) (string, error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	var sessionUrl string
	if emailRegex.MatchString(id) {
		self.Provider = prot.IdentityProvider_LINE
		sessionUrl = LINE_SESSION_LINE_URL
	} else {
		self.Provider = prot.IdentityProvider_NAVER_KR
		sessionUrl = LINE_SESSION_NAVER_URL
	}
	jsonMap, err := getJson(sessionUrl, self.header)
	if err != nil {
		return "", err
	}
	sessionKey := jsonMap["session_key"].(string)
	message := strconv.Itoa(len(sessionKey)) + sessionKey +
		strconv.Itoa(len(id)) + id +
		strconv.Itoa(len(password)) + password

	rsaKey := strings.Split(jsonMap["rsa_key"].(string), ",")
	nHex, err := hex.DecodeString(rsaKey[1])
	if err != nil {
		return "", err
	}
	n := big.NewInt(0)
	n.SetBytes(nHex)

	e, err := strconv.ParseInt(rsaKey[2], 16, 0)
	if err != nil {
		return "", err
	}

	crypto, err := rsa.EncryptPKCS1v15(rand.Reader, &rsa.PublicKey{n, int(e)}, []byte(message))
	if err != nil {
		return "", err
	}
	hexCrypto := hex.EncodeToString(crypto)

	msg, err := self.client.LoginWithIdentityCredentialForCertificate(self.Provider, id, password, true, self.IP, self.Hostname, hexCrypto)
	if err != nil {
		return "", err
	}
	self.header.Add("X-Line-Access", msg.Verifier)
	return msg.PinCode, nil
}

func (self *LineClient) GetAuthTokenAfterVerify() (string, error) {
	self.lock.Lock()
	self.lock.Unlock()
	jsonMap, err := getJson(LINE_CERTIFICATE_URL, self.header)
	if err != nil {
		return "", err
	}
	verifier := jsonMap["result"].(map[string]interface{})["verifier"].(string)
	msg, err := self.client.LoginWithVerifierForCerificate(verifier)
	if err != nil {
		return "", err
	}
	switch msg.TypeA1 {
	case prot.LoginResultType_SUCCESS:
		self.AuthToken = msg.AuthToken
	case prot.LoginResultType_REQUIRE_QRCODE:
		err = errors.New(msg.TypeA1.String())
	case prot.LoginResultType_REQUIRE_DEVICE_CONFIRM:
		err = errors.New(msg.TypeA1.String())
	}
	if err != nil {
		return "", err
	}
	return self.AuthToken, nil
}
