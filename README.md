# FirebirdFinance Assets Info

![Check](https://github.com/trustwallet/assets/workflows/Check/badge.svg)

## How to add token

Please note that __brand new tokens are not accepted__,
the projects have to be sound, with information available, and __non-minimal circulation__
(for limit details see <https://developer.trustwallet.com/assets/requirements>).

### Assets App

The [Assets web app](https://assets.trustwallet.com) can be used for most new token additions (Github account is needed).

### Quick starter

Details of the repository structure and contribution guidelines are listed on the
[Developers site](https://developer.trustwallet.com/assets/new-asset).
Here is a quick starter summary for the most common use case.


## Documentation

For details, see the [Developers site](https://developer.trustwallet.com):

- [Contribution guidelines](https://developer.trustwallet.com/assets/repository_details)

- [FAQ](https://developer.trustwallet.com/assets/faq)

## Scripts

There are several scripts available for maintainers:

- `make check` -- Execute validation checks; also used in continuous integration.
- `make fix` -- Perform automatic fixes where possible
- `make update-auto` -- Run automatic updates from external sources, executed regularly (GitHub action)
- `make add-token asset_id=c60_t0x4Fabb145d64652a948d72533023f6E7A623C7C53` -- Create `info.json` file as asset template.
- `make add-tokenlist asset_id=c60_t0x4Fabb145d64652a948d72533023f6E7A623C7C53` -- Adds a token to tokenlist.json.
- `make add-tokenlist-extended asset_id=c60_t0x4Fabb145d64652a948d72533023f6E7A623C7C53` -- Adds a token to tokenlist-extended.json.

## On Checks

This repo contains a set of scripts for verification of all the information. Implemented as Golang scripts, available through `make check`, and executed in CI build; checks the whole repo.
There are similar check logic implemented:

- in assets-management app; for checking changed token files in PRs, or when creating a PR.  Checks diffs, can be run from browser environment.
- in merge-fee-bot, which runs as a GitHub app shows result in PR comment. Executes in a non-browser environment.

## Trading pair maintenance

Info on supported trading pairs are stored in `tokenlist.json` files.
Trading pairs can be updated --
from Uniswap/Ethereum and PancakeSwap/Smartchain -- using update script (and checking in changes).
Minimal limit values for trading pair inclusion are set in the [config file](https://github.com/trustwallet/assets/blob/master/.github/assets.config.yaml).
There are also options for force-include and force-exclude in the config.

## Disclaimer

Trust Wallet team allows anyone to submit new assets to this repository. However, this does not mean that we are in direct partnership with all of the projects.

Trust Wallet team will reject projects that are deemed as scam or fraudulent after careful review.
Trust Wallet team reserves the right to change the terms of asset submissions at any time due to changing market conditions, risk of fraud, or any other factors we deem relevant.
