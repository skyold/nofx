package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"nofx/trader/types"
)

type TraderTestSuite struct {
	T       *testing.T
	Trader  interface{}
}

func NewTraderTestSuite(t *testing.T, trader interface{}) *TraderTestSuite {
	return &TraderTestSuite{
		T:      t,
		Trader: trader,
	}
}

func (s *TraderTestSuite) Cleanup() {
}

func (s *TraderTestSuite) RunAllTests() {
	s.T.Run("GetAccountInfo", func(t *testing.T) { s.TestGetAccountInfo() })
	s.T.Run("GetPositions", func(t *testing.T) { s.TestGetPositions() })
	s.T.Run("GetMarketPrice", func(t *testing.T) { s.TestGetMarketPrice() })
	s.T.Run("OpenLong", func(t *testing.T) { s.TestOpenLong() })
	s.T.Run("OpenShort", func(t *testing.T) { s.TestOpenShort() })
	s.T.Run("CloseLong", func(t *testing.T) { s.TestCloseLong() })
	s.T.Run("CloseShort", func(t *testing.T) { s.TestCloseShort() })
	s.T.Run("SetLeverage", func(t *testing.T) { s.TestSetLeverage() })
	s.T.Run("CancelAllOrders", func(t *testing.T) { s.TestCancelAllOrders() })
}

func (s *TraderTestSuite) TestGetAccountInfo() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.GetAccountInfo(ctx)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.GetBalance()
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestGetPositions() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.GetPositions(ctx)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.GetPositions()
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestGetMarketPrice() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.GetMarketPrice(ctx, "BTCUSDT")
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.GetMarketPrice("BTCUSDT")
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestOpenLong() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.OpenLong(ctx, "BTCUSDT", 0.01, 10)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.OpenLong("BTCUSDT", 0.01, 10)
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestOpenShort() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.OpenShort(ctx, "BTCUSDT", 0.01, 10)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.OpenShort("BTCUSDT", 0.01, 10)
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestCloseLong() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.CloseLong(ctx, "BTCUSDT", 0.01)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.CloseLong("BTCUSDT", 0.01)
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestCloseShort() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		_, err := trader.CloseShort(ctx, "BTCUSDT", 0.01)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		_, err := trader.CloseShort("BTCUSDT", 0.01)
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestSetLeverage() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		err := trader.SetLeverage(ctx, "BTCUSDT", 10)
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		err := trader.SetLeverage("BTCUSDT", 10)
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}

func (s *TraderTestSuite) TestCancelAllOrders() {
	ctx := context.Background()
	switch trader := s.Trader.(type) {
	case types.Trader:
		err := trader.CancelAllOrders(ctx, "BTCUSDT")
		assert.NoError(s.T, err)
	case types.ExchangeAdapter:
		err := trader.CancelAllOrders("BTCUSDT")
		assert.NoError(s.T, err)
	default:
		s.T.Skipf("Trader type %T does not implement expected interface", s.Trader)
	}
}
