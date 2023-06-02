# Changelog


## [1.0.1](https://github.com/archway-network/endpoint-controller/compare/v1.0.0...v1.0.1) (2023-06-02)


### Bug Fixes

* docker buildx failing due to environment issues ([f69bb18](https://github.com/archway-network/endpoint-controller/commit/f69bb18e2cbd308386a2524a74437b7be04c81b4))
* docker images are failing ([e560448](https://github.com/archway-network/endpoint-controller/commit/e560448857bdd5710b8337f8bc16ab2f76a89e28))
* rework dockerfile so that we only build with goreleaser ([e246a7a](https://github.com/archway-network/endpoint-controller/commit/e246a7a0d71f91da6643bc23f44730dc38a738e0))

## 1.0.0 (2023-06-01)


### Features

* .gitignore ([5a431c2](https://github.com/archway-network/endpoint-controller/commit/5a431c29663f90aa959e7361b85b4b22476cac73))
* add blockchain tests ([47fd70d](https://github.com/archway-network/endpoint-controller/commit/47fd70d1e2b07f5fb12e0a23eb41685329479346))
* add controller tests ([6d728b9](https://github.com/archway-network/endpoint-controller/commit/6d728b9e37b1578a900edd3c88422331fcb5a48c))
* add go tests to workflow ([3af65d5](https://github.com/archway-network/endpoint-controller/commit/3af65d5106a2adfa68fac4c79ba806fe21da776e))
* add utils tests ([bc08607](https://github.com/archway-network/endpoint-controller/commit/bc08607b2979fe31d9360c76b98c410a97e207d6))
* blockchain health check ([91a6a52](https://github.com/archway-network/endpoint-controller/commit/91a6a523b8489f403bd2ea637f544fb6f3180d6d))
* dockerfile ([7647a2f](https://github.com/archway-network/endpoint-controller/commit/7647a2f6d5da2b7afeeacfa7e61d25652cfc7104))
* goreleaser configuration ([d91870f](https://github.com/archway-network/endpoint-controller/commit/d91870fb9777d937c8c9a50725c069336eafdec4))
* initial version of the controller ([b6022fb](https://github.com/archway-network/endpoint-controller/commit/b6022fb31f14e6df06072e4dd837fe2223ae2576))
* kubernetes test manifests ([d9b5c9a](https://github.com/archway-network/endpoint-controller/commit/d9b5c9af84d4cd2f1f102cd9181b6aebbfac8bf0))
* Makefile ([8ed3a91](https://github.com/archway-network/endpoint-controller/commit/8ed3a91ae90ca2485a6cfea9e61acacdce5bd806))
* move checks to one function EndpointUpdateNeeded and add tests to it ([026a3e5](https://github.com/archway-network/endpoint-controller/commit/026a3e5b6e429547800489a089faa1afe87d644f))
* pr-validation workflow ([3a47dbe](https://github.com/archway-network/endpoint-controller/commit/3a47dbe36292eba376e0ade1f49b40e46e1aa9d6))
* README.md ([a7d6a86](https://github.com/archway-network/endpoint-controller/commit/a7d6a86289ce77f7aa8b67753924db71791c2c11))
* release workflow ([eba3fd5](https://github.com/archway-network/endpoint-controller/commit/eba3fd553bf55d98a24454b74137423659ff5175))
* remove tests for now until repo is public ([415acf4](https://github.com/archway-network/endpoint-controller/commit/415acf4cb21f0eddf3a240584793149ec47e6746))


### Bug Fixes

* break the loop since once its updated it does not need to be updated again ([7541476](https://github.com/archway-network/endpoint-controller/commit/75414764c3c071c261c95aebb0b22742e14e5063))
* code comment ([06cf193](https://github.com/archway-network/endpoint-controller/commit/06cf19348a0e3c9489b32e5395cc1d6858c5d749))
* correcting the logic in missed blocks check ([7d64168](https://github.com/archway-network/endpoint-controller/commit/7d6416859c1f0d8c41fbfe8e4516812225398c0a))
* do simple tcp check to avoid false alerts on grpc ([08bc530](https://github.com/archway-network/endpoint-controller/commit/08bc530ace99d9419799281ee6a2fa6eef279ec6))
* make annotations as constants ([adf1b8f](https://github.com/archway-network/endpoint-controller/commit/adf1b8f8d3a6d27b72e0cbad6d2f62b30b233ac1))
* remove unused workques and watchers ([d3e8b63](https://github.com/archway-network/endpoint-controller/commit/d3e8b63883cef372482fa6c306a88538f9878b2e))
* removing unused linter ([cf32874](https://github.com/archway-network/endpoint-controller/commit/cf32874adc04b18708bb3caacfc1775de346c8bd))
* rename variable ([b643538](https://github.com/archway-network/endpoint-controller/commit/b643538838097fd754b13bf4b54bcb94b95db4d7))
* return early if there isnt any healthy targets ([1b4d658](https://github.com/archway-network/endpoint-controller/commit/1b4d658b33daadca81a64bae1c27fedb2a40bb6b))
* set http timeout ([7c6e1a7](https://github.com/archway-network/endpoint-controller/commit/7c6e1a7bcb286bb0e0c374a60f5731d4a4df0fe0))
* trim the target annotation from spaces ([1dfefe6](https://github.com/archway-network/endpoint-controller/commit/1dfefe649adbcf4c0590a49320f18217eaf9774f))
* typo fix on the watcher ([18cb450](https://github.com/archway-network/endpoint-controller/commit/18cb45046ad78a58c4c07c72abbac4d618906743))
* use block update instead of handling one by one ([67bf98b](https://github.com/archway-network/endpoint-controller/commit/67bf98b0fc564aedc416a169339150f4bf123b0c))
* use services namespace when checking endpoints ([90d54ec](https://github.com/archway-network/endpoint-controller/commit/90d54ec1d8ad6a31ed93637bbf525a735cf0dcc8))
