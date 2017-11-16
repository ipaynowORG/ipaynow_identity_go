package identity

import (
	"bytes"
	"crypto/des"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type App struct {
	AppId  string
	AppKey string
	DesKey string
	IsDev  bool
}

/**
     * 身份验证
	 * @param app appId(应用ID)和appKey ,desKey
     * @param cardName  姓名
     * @param idcard    身份证
     * @param mhtOrderNo    商户订单号(可空,为空时自动生成)
     * @return
*/
func IdentityAuth(app *App, cardName string, idcard string, mhtOrderNo string) string {

	var postMap = make(map[string]string)

	postMap["cardName"] = cardName
	postMap["idcard"] = idcard
	if mhtOrderNo != "" {
		postMap["mhtOrderNo"] = mhtOrderNo
	} else {
		postMap["mhtOrderNo"] = getRandomString(20)
	}

	return query(app, postMap, "ID01")
}

/**
 * 身份验证-订单查询
 * @param app appId(应用ID)和appKey ,desKey
 * @param mhtOrderNo    商户订单号
 * @return
 */
func IdentityAuthQuery(app *App, mhtOrderNo string) string {

	var postMap = make(map[string]string)
	postMap["mhtOrderNo"] = mhtOrderNo

	return query(app, postMap, "ID01_Query")
}

/**
     *  卡信息认证
	 * @param app appId(应用ID)和appKey ,desKey
     * @param idCardName   姓名
     * @param idCard    身份证
     * @param bankCardNum   银行账户
     * @param mhtOrderNo    商户订单号(可空,为空时自动生成)
     * @return
*/
func CardAuth(app *App, idCardName string, idCard string, bankCardNum string, mhtOrderNo string) string {

	var postMap = make(map[string]string)
	postMap["idCardName"] = idCardName
	postMap["idCard"] = idCard
	postMap["bankCardNum"] = bankCardNum
	if mhtOrderNo != "" {
		postMap["mhtOrderNo"] = mhtOrderNo
	} else {
		postMap["mhtOrderNo"] = getRandomString(20)
	}
	return query(app, postMap, "ID02")
}

/**
     * 卡信息认证- 订单查询
	 * @param app appId(应用ID)和appKey ,desKey
     * @param mhtOrderNo
     * @return
*/
func CardAuthQuery(app *App, mhtOrderNo string) string {

	var postMap = make(map[string]string)
	postMap["mhtOrderNo"] = mhtOrderNo
	return query(app, postMap, "ID02_Query")
}

/**
     * 手机号认证
	 * @param app appId(应用ID)和appKey ,desKey
     * @param idCardName    认证姓名
     * @param idCard    身份证号码
     * @param mobile    手机号
     * @param mhtOrderNo    商户订单号
     * @return
*/
func MobileNoAuth(app *App, idCardName string, idCard string, mobile string, mhtOrderNo string) string {

	var postMap = make(map[string]string)
	postMap["idCardName"] = idCardName
	postMap["idCard"] = idCard
	postMap["mobile"] = mobile
	if mhtOrderNo != "" {
		postMap["mhtOrderNo"] = mhtOrderNo
	} else {
		postMap["mhtOrderNo"] = getRandomString(20)
	}
	return query(app, postMap, "ID03")
}

/**
 * 手机号认证 - 订单查询
 * @param mhtOrderNo
 * @return
 */
func MobileNoAuthQuery(app *App, mhtOrderNo string) string {
	var postMap = make(map[string]string)
	postMap["mhtOrderNo"] = mhtOrderNo
	return query(app, postMap, "ID03_Query")
}

func query(app *App, postMap map[string]string, funcode string) string {

	//2. map 2 kv 并排序
	var keys []string
	for k := range postMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var postFormLinkReport = ""
	for _, k := range keys {
		postFormLinkReport += k + "=" + postMap[k] + "&"
	}
	postFormLinkReport = postFormLinkReport[0 : len(postFormLinkReport)-1]
	//3. message=base64(appId=xxx)| base64(3DES(报文原文))|base64(MD5(报文原文+&+ md5Key))
	var message1 = "appId=" + app.AppId
	b64 := base64.StdEncoding.EncodeToString([]byte(message1))
	message1 = string(b64)
	var des, err = tripleEcbDesEncrypt([]byte(postFormLinkReport), []byte(app.DesKey))
	if err != nil {
		fmt.Println(des)
		fmt.Println(err)
	}
	var message2 = base64.StdEncoding.EncodeToString([]byte(des))
	var tmp = fmt.Sprintf("%x", md5.Sum([]byte(postFormLinkReport+"&"+app.AppKey)))
	var message3 = base64.StdEncoding.EncodeToString([]byte(tmp))
	var message = message1 + "|" + message2 + "|" + message3 + ""

	//4. urlencoder
	u := url.Values{}
	u.Set("message", message)

	//5. post funcode=xxx&message=xxx
	var url = ""
	if app.IsDev {
		url = "https://dby.ipaynow.cn/identify"
	} else {
		url = "https://s.ipaynow.cn/auth"
	}
	var result = post(url, "funcode="+funcode+"&"+u.Encode())

	//6.基本验证
	if len(strings.Split(result, "|")) == 2 {
		decodeBytes, err := base64.StdEncoding.DecodeString(strings.Split(result, "|")[1])
		if err == nil {
			fmt.Println(string(decodeBytes))
		} else {
			fmt.Println(err)
		}
	}

	//7.解析
	//	return1 := strings.Split(result, "|")[0]
	return2 := strings.Split(result, "|")[1]
	return3 := strings.Split(result, "|")[2]

	return2b64, err2 := base64.StdEncoding.DecodeString(return2)
	if err2 == nil {
		var originalMsg, err1 = tripleEcbDesDecrypt([]byte(return2b64), []byte(app.DesKey))
		if err1 == nil {
			//验签
			var mySign = fmt.Sprintf("%x", md5.Sum([]byte(string(originalMsg)+"&"+app.AppKey)))
			var originalSign, err4 = base64.StdEncoding.DecodeString(return3)
			if err4 == nil {
				//验签失败?
				if string(originalSign) != mySign {
					return string(originalMsg)
				}
				return string(originalMsg)
			} else {
				fmt.Println(err4)
			}
		} else {
			fmt.Println(err1)
		}
	} else {
		fmt.Println(err2)
	}
	return ""
}

func urlEncode(content string) string {
	l, e := url.Parse("?" + content)
	if e != nil {
		fmt.Println(l, e)
	}
	return l.Query().Encode()[0 : len(l.Query().Encode())-1]
}

func post(url string, postcontent string) string {
	resp, err := http.Post(url,
		"application/x-www-form-urlencoded",
		strings.NewReader(postcontent))
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// handle error
	}
	return string(body)
}

func getRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

//ECB PKCS5Padding
func pKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func noPadding(origData []byte) []byte {
	length := len(origData)

	if length%8 != 0 {
		var len = length - length%8 + 8
		var needData = make([]byte, len)
		for i := 0; i < len; i++ {
			needData[i] = 0x00
		}
		copy(needData, origData)
		return needData
	} else {
		return origData
	}
}

//ECB PKCS5Unpadding
func pKCS5Unpadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//Des加密
func encrypt(origData, key []byte) ([]byte, error) {
	if len(origData) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(origData)%bs != 0 {
		return nil, errors.New("wrong padding")
	}
	out := make([]byte, len(origData))
	dst := out
	for len(origData) > 0 {
		block.Encrypt(dst, origData[:bs])
		origData = origData[bs:]
		dst = dst[bs:]
	}
	return out, nil
}

//Des解密
func decrypt(crypted, key []byte) ([]byte, error) {
	if len(crypted) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(crypted))
	dst := out
	bs := block.BlockSize()
	if len(crypted)%bs != 0 {
		return nil, errors.New("wrong crypted size1" + string(bs) + "_" + string(len(crypted)))
	}

	for len(crypted) > 0 {
		block.Decrypt(dst, crypted[:bs])
		crypted = crypted[bs:]
		dst = dst[bs:]
	}

	return out, nil
}

//[golang ECB 3DES Encrypt]
func tripleEcbDesEncrypt(origData, key []byte) ([]byte, error) {
	tkey := make([]byte, 24, 24)
	copy(tkey, key)
	k1 := tkey[:8]
	k2 := tkey[8:16]
	k3 := tkey[16:]

	//	block, err := des.NewCipher(k1)
	//	if err != nil {
	//		return nil, err
	//	}
	//	bs := block.BlockSize()
	origData = noPadding(origData)

	buf1, err := encrypt(origData, k1)
	if err != nil {
		return nil, err
	}
	buf2, err := decrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := encrypt(buf2, k3)
	if err != nil {
		return nil, err
	}
	return out, nil
}

//[golang ECB 3DES Decrypt]
func tripleEcbDesDecrypt(crypted, key []byte) ([]byte, error) {
	tkey := make([]byte, 24, 24)
	copy(tkey, key)
	k1 := tkey[:8]
	k2 := tkey[8:16]
	k3 := tkey[16:]
	buf1, err := decrypt(crypted, k3)
	if err != nil {
		return nil, err
	}
	buf2, err := encrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := decrypt(buf2, k1)
	if err != nil {
		return nil, err
	}
	out = pKCS5Unpadding(out)
	return out, nil
}
