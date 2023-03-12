package api_test

/* import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stratumfarm/go-miningcore-client"
	"github.com/1oopio/phantomias/api"
	_ "github.com/1oopio/phantomias/cmd"
	"github.com/1oopio/phantomias/config"
	"github.com/1oopio/phantomias/price"
	"github.com/stretchr/testify/assert"
)

var (
	testServer *httptest.Server
	apiServer  *api.Server
)

type testHandler struct {
	route string
	file  string
}

func TestMain(m *testing.M) {
	handler := http.NewServeMux()

	for _, h := range []testHandler{
		{route: "/api/pools", file: "./testdata/pools.json"},
		{route: "/api/pools/eth1", file: "testdata/pools_eth1.json"},
		{route: "/api/v2/pools/eth1/blocks", file: "testdata/pools_eth1_blocks.json"},
		{route: "/api/v2/pools/eth1/payments", file: "testdata/pools_eth1_payments.json"},
		{route: "/api/pools/eth1/performance", file: "testdata/pools_eth1_performance.json"},
		{route: "/api/pools/eth1/miners", file: "testdata/pools_eth1_miners.json"},
		{route: "/api/pools/eth1/miners/0xd0b706c48078ee87db9d0bef92453a66b1ab9d44", file: "testdata/pools_eth1_miners_0xd0b706c48078ee87db9d0bef92453a66b1ab9d44.json"},
		{route: "/api/v2/pools/eth1/miners/0x017b67b81340634bbc2145946a9e99c63dd9696c/payments", file: "testdata/pools_eth1_miners_0x017b67b81340634bbc2145946a9e99c63dd9696c_payments.json"},
		{route: "/api/v2/pools/eth1/miners/0x017b67b81340634bbc2145946a9e99c63dd9696c/balancechanges", file: "testdata/pools_eth1_miners_0x017b67b81340634bbc2145946a9e99c63dd9696c_balancechanges.json"},
		{route: "/api/v2/pools/eth1/miners/0x017b67b81340634bbc2145946a9e99c63dd9696c/earnings/daily", file: "testdata/pools_eth1_miners_0x017b67b81340634bbc2145946a9e99c63dd9696c_earnings_daily.json"},
	} {
		handler.HandleFunc(h.route, func(route, file string) func(w http.ResponseWriter, r *http.Request) {
			return func(w http.ResponseWriter, r *http.Request) {
				data, err := os.ReadFile(file)
				if err != nil {
					w.WriteHeader(fiber.StatusInternalServerError)
					return
				}
				w.Write(data)
			}
		}(h.route, h.file))
	}

	testServer = httptest.NewServer(handler)
	defer testServer.Close()

	cfg, err := config.Load("")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	apiServer = api.New(context.Background(), cfg.Proxy, cfg.Pools, newClient(), nil, &priceClient{}, nil)

	code := m.Run()
	os.Exit(code)
}

func newClient() *miningcore.Client {
	return miningcore.New(
		testServer.URL,
	)
}

type priceClient struct{}

func (p *priceClient) Start(_ ...time.Duration) {}
func (p *priceClient) Close()                   {}
func (p *priceClient) LoadPrices() error        { return nil }
func (p *priceClient) GetPrices(_ string) []*price.Price {
	return []*price.Price{
		{
			Coin:                     "ethereum",
			VSCurrency:               "chf",
			Price:                    1612.88,
			PriceChangePercentage24H: 2.3,
		},
	}
}

type handlerTest struct {
	description string

	// Test input
	route string

	// Expected output
	expectedError bool
	expectedCode  int
	compareBody   func(*testing.T, []byte)
}

func TestHandlers(t *testing.T) {
	tests := []*handlerTest{
		{
			description:   "not found",
			route:         "/notfound",
			expectedError: false,
			expectedCode:  fiber.StatusNotFound,
		},
		{
			description:   "get all pools",
			route:         "/api/v1/pools",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.PoolsRes{}, &api.PoolsRes{}, "testdata/proxy/pools.json"),
		},
		{
			description:   "get pool eth1",
			route:         "/api/v1/pools/eth1",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.PoolExtendedRes{}, &api.PoolExtendedRes{}, "testdata/proxy/pools_eth1.json"),
		},
		{
			description:   "get pool eth1 blocks",
			route:         "/api/v1/pools/eth1/blocks",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.BlocksRes{}, &api.BlocksRes{}, "testdata/proxy/pools_eth1_blocks.json"),
		},
		{
			description:   "get pool eth1 payments",
			route:         "/api/v1/pools/eth1/payments",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.PaymentsRes{}, &api.PaymentsRes{}, "testdata/proxy/pools_eth1_payments.json"),
		},
		{
			description:   "get pool eth1 performance",
			route:         "/api/v1/pools/eth1/performance",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.PoolPerformanceRes{}, &api.PoolPerformanceRes{}, "testdata/proxy/pools_eth1_performance.json"),
		},
		{
			description:   "get pool eth1 miners",
			route:         "/api/v1/pools/eth1/miners",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.MinersRes{}, &api.MinersRes{}, "testdata/proxy/pools_eth1_miners.json"),
		},
		{
			description:   "get pool eth1 miner 0xd0b706c48078ee87db9d0bef92453a66b1ab9d44",
			route:         "/api/v1/pools/eth1/miners/0xd0b706c48078ee87db9d0bef92453a66b1ab9d44",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.MinerRes{}, &api.MinerRes{}, "testdata/proxy/pools_eth1_miners_0xd0b706c48078ee87db9d0bef92453a66b1ab9d44.json"),
		},
		{
			description:   "get pool eth1 miner 0x017b67b81340634bbc2145946a9e99c63dd9696c payments",
			route:         "/api/v1/pools/eth1/miners/0x017b67b81340634bbc2145946a9e99c63dd9696c/payments",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.PaymentsRes{}, &api.PaymentsRes{}, "testdata/proxy/pools_eth1_miners_0x017b67b81340634bbc2145946a9e99c63dd9696c_payments.json"),
		},
		{
			description:   "get pool eth1 miner 0x017b67b81340634bbc2145946a9e99c63dd9696c balancechanges",
			route:         "/api/v1/pools/eth1/miners/0x017b67b81340634bbc2145946a9e99c63dd9696c/balancechanges",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.BalanceChangesRes{}, &api.BalanceChangesRes{}, "testdata/proxy/pools_eth1_miners_0x017b67b81340634bbc2145946a9e99c63dd9696c_balancechanges.json"),
		},
		{
			description:   "get pool eth1 miner 0x017b67b81340634bbc2145946a9e99c63dd9696c daily earnings",
			route:         "/api/v1/pools/eth1/miners/0x017b67b81340634bbc2145946a9e99c63dd9696c/earnings/daily",
			expectedError: false,
			expectedCode:  200,
			compareBody:   defaultCompareBody(t, &api.DailyEarningRes{}, &api.DailyEarningRes{}, "testdata/proxy/pools_eth1_miners_0x017b67b81340634bbc2145946a9e99c63dd9696c_earnings_daily.json"),
		},
	}

	// Iterate through test single test cases
	for _, test := range tests {
		t.Run(
			test.description,
			func(t *testing.T) {
				// Create a new http request with the route
				// from the test case
				req, _ := http.NewRequest(
					"GET",
					test.route,
					nil,
				)

				// Perform the request plain with the app.
				// The -1 disables request latency.
				res, err := apiServer.API().Test(req, -1)
				defer res.Body.Close()

				// verify that no error occured, that is not expected
				assert.Equalf(t, test.expectedError, err != nil, test.description)

				// Verify if the status code is as expected
				assert.Equalf(t, test.expectedCode, res.StatusCode, test.description)

				// As expected errors lead to broken responses, the next
				// test case needs to be processed
				if test.expectedError {
					return
				}

				// Read the response body
				body, err := ioutil.ReadAll(res.Body)

				// Reading the response body should work everytime, such that
				// the err variable should be nil
				assert.NoErrorf(t, err, test.description)

				// Verify, that the reponse body equals the expected body
				if test.compareBody != nil {
					test.compareBody(t, body)
				}
			},
		)
	}
}

func defaultCompareBody(t *testing.T, res, exp any, file string) func(t *testing.T, body []byte) {
	return func(t *testing.T, body []byte) {
		err := json.Unmarshal(body, &res)
		assert.NoError(t, err)

		err = json.Unmarshal(mustFile(file), &exp)
		assert.NoError(t, err)
		assert.Equal(t, exp, res)
	}
}

func mustFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}
*/
