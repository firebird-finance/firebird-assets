package processor

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	fileLib "github.com/trustwallet/assets-go-libs/file"
	"github.com/trustwallet/assets-go-libs/image"
	"github.com/trustwallet/assets-go-libs/path"
	"github.com/trustwallet/assets-go-libs/validation/info"
	"github.com/trustwallet/assets-go-libs/validation/tokenlist"
	"github.com/trustwallet/go-libs/blockchain/binance"
	"github.com/trustwallet/go-libs/blockchain/binance/explorer"
	assetlib "github.com/trustwallet/go-primitives/asset"
	"github.com/trustwallet/go-primitives/coin"
	"github.com/trustwallet/go-primitives/numbers"
	"github.com/trustwallet/go-primitives/types"

	"github.com/trustwallet/assets/internal/config"
)

const (
	assetsPage       = 1
	assetsRows       = 1000
	marketPairsLimit = 1000
	tokensListLimit  = 10000

	activeStatus = "active"
)

func (s *Service) UpdateBinanceTokens() error {
	explorerClient := explorer.InitClient(config.Default.ClientURLs.Binance.Explorer, nil)

	bep2AssetList, err := explorerClient.FetchBep2Assets(assetsPage, assetsRows)
	if err != nil {
		return err
	}

	dexClient := binance.InitClient(config.Default.ClientURLs.Binance.Dex, "", nil)

	marketPairs, err := dexClient.FetchMarketPairs(marketPairsLimit)
	if err != nil {
		return err
	}

	tokenList, err := dexClient.FetchTokens(tokensListLimit)
	if err != nil {
		return err
	}

	chain, err := types.GetChainFromAssetType(string(types.BEP2))
	if err != nil {
		return err
	}

	err = fetchMissingAssets(chain, bep2AssetList.AssetInfoList)
	if err != nil {
		return err
	}

	tokens, err := generateTokenList(marketPairs, tokenList)
	if err != nil {
		return err
	}

	sortTokens(tokens)

	return createTokenListJSON(chain, tokens)
}

func fetchMissingAssets(chain coin.Coin, assets []explorer.Bep2Asset) error {
	for _, a := range assets {
		if a.AssetImg == "" || a.Decimals == 0 {
			continue
		}

		assetLogoPath := path.GetAssetLogoPath(chain.Handle, a.Asset)
		if fileLib.Exists(assetLogoPath) {
			continue
		}

		if err := createLogo(assetLogoPath, a); err != nil {
			return err
		}

		if err := createInfoJSON(chain, a); err != nil {
			return err
		}
	}

	return nil
}

func createLogo(assetLogoPath string, a explorer.Bep2Asset) error {
	err := fileLib.CreateDirPath(assetLogoPath)
	if err != nil {
		return err
	}

	return image.CreatePNGFromURL(a.AssetImg, assetLogoPath)
}

func createInfoJSON(chain coin.Coin, a explorer.Bep2Asset) error {
	explorerURL, err := coin.GetCoinExploreURL(chain, a.Asset, "")
	if err != nil {
		return err
	}

	assetType := string(types.BEP2)
	website := ""
	description := "-"
	status := activeStatus

	assetInfo := info.AssetModel{
		Name:        &a.Name,
		Type:        &assetType,
		Symbol:      &a.MappedAsset,
		Decimals:    &a.Decimals,
		Website:     &website,
		Description: &description,
		Explorer:    &explorerURL,
		Status:      &status,
		ID:          &a.Asset,
	}

	assetInfoPath := path.GetAssetInfoPath(chain.Handle, a.Asset)

	data, err := fileLib.PrepareJSONData(&assetInfo)
	if err != nil {
		return err
	}

	return fileLib.CreateJSONFile(assetInfoPath, data)
}

func createTokenListJSON(chain coin.Coin, tokens []tokenlist.Token) error {
	tokenListPath := path.GetTokenListPath(chain.Handle, path.TokenlistDefault)

	var oldTokenList tokenlist.Model
	err := fileLib.ReadJSONFile(tokenListPath, &oldTokenList)
	if err != nil {
		return nil
	}

	if reflect.DeepEqual(oldTokenList.Tokens, tokens) {
		return nil
	}

	if len(tokens) == 0 {
		return nil
	}

	data, err := fileLib.PrepareJSONData(&tokenlist.Model{
		Name:      fmt.Sprintf("Trust Wallet: %s", coin.Coins[chain.ID].Name),
		LogoURI:   config.Default.URLs.Logo,
		Timestamp: time.Now().Format(config.Default.TimeFormat),
		Tokens:    tokens,
		Version:   tokenlist.Version{Major: oldTokenList.Version.Major + 1},
	})
	if err != nil {
		return err
	}

	log.Debugf("Tokenlist: list with %d tokens and %d pairs written to %s.",
		len(tokens), countTotalPairs(tokens), tokenListPath)

	return fileLib.CreateJSONFile(tokenListPath, data)
}

func countTotalPairs(tokens []tokenlist.Token) int {
	var counter int
	for _, token := range tokens {
		counter += len(token.Pairs)
	}

	return counter
}

func sortTokens(tokens []tokenlist.Token) {
	sort.Slice(tokens, func(i, j int) bool {
		if len(tokens[i].Pairs) != len(tokens[j].Pairs) {
			return len(tokens[i].Pairs) > len(tokens[j].Pairs)
		}

		return tokens[i].Address < tokens[j].Address
	})

	for _, token := range tokens {
		sort.Slice(token.Pairs, func(i, j int) bool {
			return token.Pairs[i].Base < token.Pairs[j].Base
		})
	}
}

func generateTokenList(marketPairs []binance.MarketPair, tokenList binance.Tokens) ([]tokenlist.Token, error) {
	if len(marketPairs) < 5 {
		return nil, fmt.Errorf("no markets info is returned from Binance DEX: %d", len(marketPairs))
	}

	if len(tokenList) < 5 {
		return nil, fmt.Errorf("no tokens info is returned from Binance DEX: %d", len(tokenList))
	}

	pairsMap := make(map[string][]tokenlist.Pair)
	pairsList := make(map[string]struct{})
	tokensMap := make(map[string]binance.Token)

	for _, token := range tokenList {
		tokensMap[token.Symbol] = token
	}

	for _, marketPair := range marketPairs {
		if !isTokenExistOrActive(marketPair.BaseAssetSymbol) || !isTokenExistOrActive(marketPair.QuoteAssetSymbol) {
			continue
		}

		tokenSymbol := marketPair.QuoteAssetSymbol

		if val, exists := pairsMap[tokenSymbol]; exists {
			val = append(val, getPair(marketPair))
			pairsMap[tokenSymbol] = val
		} else {
			pairsMap[tokenSymbol] = []tokenlist.Pair{getPair(marketPair)}
		}

		pairsList[marketPair.BaseAssetSymbol] = struct{}{}
		pairsList[marketPair.QuoteAssetSymbol] = struct{}{}
	}

	tokenItems := make([]tokenlist.Token, 0, len(pairsList))

	for pair := range pairsList {
		token := tokensMap[pair]

		tokenItems = append(tokenItems, tokenlist.Token{
			Asset:    getAssetIDSymbol(token.Symbol, coin.Coins[coin.BINANCE].Symbol, coin.BINANCE),
			Type:     getTokenType(token.Symbol, coin.Coins[coin.BINANCE].Symbol, types.BEP2),
			Address:  token.Symbol,
			Name:     getTokenName(token),
			Symbol:   token.OriginalSymbol,
			Decimals: coin.Coins[coin.BINANCE].Decimals,
			LogoURI:  getLogoURI(token.Symbol, coin.Coins[coin.BINANCE].Handle, coin.Coins[coin.BINANCE].Symbol),
			Pairs:    pairsMap[token.Symbol],
		})
	}

	return tokenItems, nil
}

func isTokenExistOrActive(symbol string) bool {
	if symbol == coin.Coins[coin.BINANCE].Symbol {
		return true
	}

	assetPath := path.GetAssetInfoPath(coin.Coins[coin.BINANCE].Handle, symbol)

	var infoAsset info.AssetModel
	if err := fileLib.ReadJSONFile(assetPath, &infoAsset); err != nil {
		log.Debug(err)
		return false
	}

	if infoAsset.GetStatus() != activeStatus {
		log.Debugf("asset status [%s] is not active", symbol)
		return false
	}

	return true
}

func getPair(marketPair binance.MarketPair) tokenlist.Pair {
	return tokenlist.Pair{
		Base:     getAssetIDSymbol(marketPair.BaseAssetSymbol, coin.Coins[coin.BINANCE].Symbol, coin.BINANCE),
		LotSize:  strconv.FormatInt(numbers.ToSatoshi(marketPair.LotSize), 10),
		TickSize: strconv.FormatInt(numbers.ToSatoshi(marketPair.TickSize), 10),
	}
}

func getAssetIDSymbol(tokenID string, nativeCoinID string, coinType uint) string {
	if tokenID == nativeCoinID {
		return assetlib.BuildID(coinType, "")
	}

	return assetlib.BuildID(coinType, tokenID)
}

func getTokenType(symbol string, nativeCoinSymbol string, tokenType types.TokenType) types.TokenType {
	if symbol == nativeCoinSymbol {
		return types.Coin
	}

	return tokenType
}

func getLogoURI(id, githubChainFolder, nativeCoinSymbol string) string {
	if id == nativeCoinSymbol {
		return path.GetChainLogoURL(config.Default.URLs.AssetsApp, githubChainFolder)
	}

	return path.GetAssetLogoURL(config.Default.URLs.AssetsApp, githubChainFolder, id)
}

func getTokenName(t binance.Token) string {
	if t.Symbol == coin.Binance().Symbol && t.Name == "Binance Chain Native Token" {
		return "BNB Beacon Chain"
	}

	return t.Name
}
