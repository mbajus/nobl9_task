package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	apiKey       = "8261ecdd-c908-445e-a2ef-81e5449ef6d5"
	randomApiUrl = "https://api.random.org/json-rpc/4/invoke"
)

var (
	answearsMu sync.Mutex
	wg         sync.WaitGroup
)

type RAdata struct {
	RandNums []float64 `json:"data"`
}

type RArandom struct {
	Randrandom RAdata `json:"random"`
}

type RAresult struct {
	Randresult RArandom `json:"result"`
}

type Answear struct {
	Ds   float64   `json:"stddev"`
	List []float64 `json:"data"`
}

func getSdev(nums []float64) (sdev float64) {
	mean := 0.0
	for _, num := range nums {
		mean += num
	}
	mean = mean / float64(len(nums))
	for _, num := range nums {
		sdev += (num - mean) * (num - mean)
	}
	sdev = sdev / float64(len(nums))
	sdev = math.Sqrt(sdev)
	return
}

func getRand(n, nums int, answears *[]Answear) {
	defer wg.Done()
	jsonReq := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "generateIntegers",
		"params": {
			"apiKey": "%s",
			"n": %d,
			"min": 1,
			"max": 1000
		},
		"id": %d
	}`, apiKey, nums, n)
	resp, err := http.Post(randomApiUrl, "application/json", strings.NewReader(jsonReq))
	if err != nil {
		log.Println("Respond failure from Random API:\n", err.Error())
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to parse request body:\n", err.Error())
	}
	var list RAresult
	if err := json.Unmarshal(body, &list); err != nil {
		fmt.Println("Error parshing json:\n", err.Error())
	}
	fmt.Println(list.Randresult.Randrandom.RandNums)
	sdev := getSdev(list.Randresult.Randrandom.RandNums)
	answearsMu.Lock()
	*answears = append(*answears, Answear{Ds: sdev, List: list.Randresult.Randrandom.RandNums})
	answearsMu.Unlock()
}

func getAnsw(c *gin.Context) {
	var (
		answears []Answear
	)
	nreq, err := strconv.Atoi(c.Query("requests"))
	if err != nil {
		c.JSON(http.StatusBadRequest, "GET request not valid, {r} is not a number.")
		return
	}
	nums, err := strconv.Atoi(c.Query("length"))
	if err != nil {
		c.JSON(http.StatusBadRequest, "GET request not valid, {l} is not a number.")
		return
	}
	fmt.Printf("GET request, %d concurrent requests with %d random integers.\n", nreq, nums)
	wg.Add(nreq)
	for i := 0; i != nreq; i++ {
		go getRand(i, nums, &answears)
	}
	wg.Wait()
	if !(len(answears) == nreq) {
		c.JSON(http.StatusInternalServerError, "The response from Random API doesn't match the requirements.")
		return
	}
	if !(len(answears) == nreq) {
		c.JSON(http.StatusInternalServerError, "The response from Random API doesn't match the requirements.")
		return
	}
	var allsets []float64
	for _, arr := range answears {
		allsets = append(allsets, arr.List...)
		if !(len(arr.List) == nums) {
			c.JSON(http.StatusInternalServerError, "The response from Random API doesn't match the requirements.")
			return
		}
	}
	answears = append(answears, Answear{Ds: getSdev(allsets), List: allsets})
	c.PureJSON(http.StatusOK, answears)
}

func main() {
	router := gin.New()
	router.GET("/random/mean", getAnsw)
	fmt.Println("Serving on: localhost:8080")
	log.Fatal(router.Run(":8080"))
}
