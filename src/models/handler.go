package models

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	URL          = "http://www.ritsumei.ac.jp/academic-affairs/status/#"
	KIC_SELECTOR = "#main > section.campusNews > section.kic.box.clearfix > div > div.campusText > div.markBox > ul > li > strong > span.eng"
	BKC_SELECTOR = "#main > section.campusNews > section.bkc.box.clearfix > div > div.campusText > div.markBox > ul > li > span > strong > span.eng"
	OIC_SELECTOR = "#main > section.campusNews > section.oic.box.clearfix > div > div.campusText > div.markBox > ul > li > strong > span.eng"
	UPDATED_AT   = "#main > div:nth-child(1) > section > div > dl > dd"
)

type ResponseData struct {
	Campus       string `json:"campus"`
	IsCancel     string `json:"isCancel"`
	UpdatedAt    string `json:"updatedAt"`
	ErrorMassage string `json:"errorMassage"`
}

var re = regexp.MustCompile(`\d{4}/\d{2}/\d{2}`)

func Log(handler http.Handler) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println(time.Now(), request.RemoteAddr, request.URL.Path, request.Body)
		defer request.Body.Close()
		handler.ServeHTTP(writer, request)
	}
}

func Handler() error {
	http.HandleFunc("/api/cancellation/", ResponseCalcellInfo())
	s := http.Server{
		Addr:    ":8086",
		Handler: Log(http.DefaultServeMux),
	}
	s.ListenAndServe()
	return nil
}

func ResponseCalcellInfo() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		// CORS setting
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Access-Control-Allow-Origin", "*")

		if request.Method != http.MethodGet {
			http.Error(writer, `{"errorMassage": "405 method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		campus := strings.Split(request.URL.Path, "/")[3]
		if !(campus == "kic" || campus == "bkc" || campus == "oic") {
			http.Error(writer, `{"errorMassage": "404 page not found"}`, http.StatusNotFound)
			return
		}
		info, err := getCancelInfo(campus)
		if err != nil {
			http.Error(writer, `{"errorMassage": "503 service unavailable"}`, http.StatusServiceUnavailable)
		}
		res, err := json.Marshal(info)

		fmt.Fprint(writer, string(res))
	}
}

// 正しいエンドポイントが指定された場合のみ呼ばれる
func getCancelInfo(campus string) (responseData ResponseData, err error) {
	// fetch cancel info page
	resp, _ := http.Get(URL)
	defer resp.Body.Close()
	html, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return responseData, err
	}
	// scraping each campus info
	responseData.Campus = campus
	switch {
	case campus == "kic":
		responseData.IsCancel = strconv.FormatBool(strings.Contains(html.Find(KIC_SELECTOR).Text(), "Canceled"))
	case campus == "bkc":
		responseData.IsCancel = strconv.FormatBool(strings.Contains(html.Find(BKC_SELECTOR).Text(), "Canceled"))
	case campus == "oic":
		responseData.IsCancel = strconv.FormatBool(strings.Contains(html.Find(OIC_SELECTOR).Text(), "Canceled"))
	}

	responseData.UpdatedAt = re.FindString(html.Find(UPDATED_AT).Text()) +
		" " + html.Find(UPDATED_AT+" > strong").
		Text()
	responseData.ErrorMassage = ""
	return responseData, nil
}
