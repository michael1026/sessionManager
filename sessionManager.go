package sessionManager

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type CookieInfo map[string]string

func readCookiesFromString(s string) []*http.Cookie {
	cookieStrings := strings.Split(s, ";")

	for i, c := range cookieStrings {
		cookieStrings[i] = strings.TrimSpace(c)
	}

	cookieCount := len(cookieStrings)
	if cookieCount == 0 {
		return []*http.Cookie{}
	}
	cookies := make([]*http.Cookie, 0, cookieCount)
	for _, line := range cookieStrings {
		parts := strings.Split(strings.TrimSpace(line), ";")
		if len(parts) == 1 && parts[0] == "" {
			continue
		}
		parts[0] = strings.TrimSpace(parts[0])
		j := strings.Index(parts[0], "=")
		if j < 0 {
			continue
		}
		name, value := parts[0][:j], parts[0][j+1:]

		value, ok := parseCookieValue(value, true)
		if !ok {
			continue
		}
		c := &http.Cookie{
			Name:  name,
			Value: value,
			Raw:   line,
		}
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.TrimSpace(parts[i])
			if len(parts[i]) == 0 {
				continue
			}

			attr, val := parts[i], ""
			if j := strings.Index(attr, "="); j >= 0 {
				attr, val = attr[:j], attr[j+1:]
			}
			lowerAttr := strings.ToLower(attr)
			val, ok = parseCookieValue(val, false)
			if !ok {
				c.Unparsed = append(c.Unparsed, parts[i])
				continue
			}
			switch lowerAttr {
			case "secure":
				c.Secure = true
				continue
			case "httponly":
				c.HttpOnly = true
				continue
			case "domain":
				c.Domain = val
				continue
			case "max-age":
				secs, err := strconv.Atoi(val)
				if err != nil || secs != 0 && val[0] == '0' {
					break
				}
				if secs <= 0 {
					secs = -1
				}
				c.MaxAge = secs
				continue
			case "expires":
				c.RawExpires = val
				exptime, err := time.Parse(time.RFC1123, val)
				if err != nil {
					exptime, err = time.Parse("Mon, 02-Jan-2006 15:04:05 MST", val)
					if err != nil {
						c.Expires = time.Time{}
						break
					}
				}
				c.Expires = exptime.UTC()
				continue
			case "path":
				c.Path = val
				continue
			}
			c.Unparsed = append(c.Unparsed, parts[i])
		}
		cookies = append(cookies, c)
	}
	return cookies
}

func parseCookieValue(raw string, allowDoubleQuote bool) (string, bool) {
	// Strip the quotes, if present.
	if allowDoubleQuote && len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	return raw, true
}

func readCookieJson(filepath string) *cookiejar.Jar {
	jar, err := cookiejar.New(nil)

	if err != nil {
		log.Fatal("Error reading cookie file")
	}

	if filepath == "" {

		return jar
	}
	cookieFile, err := os.Open(filepath)
	var cookies CookieInfo

	if err != nil {
		log.Fatal("Error creating cookie jar")
	}

	defer cookieFile.Close()

	bytes, _ := ioutil.ReadAll(cookieFile)

	json.Unmarshal(bytes, &cookies)

	for rawUrl, cookieString := range cookies {
		parsedUrl, err := url.Parse(rawUrl)

		if err != nil {
			continue
		}

		jar.SetCookies(parsedUrl, readCookiesFromString(cookieString))
	}

	return jar
}
