// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT License was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// This file defines the Node.js v24.6.0 TLS ClientHelloSpec and provides
// fingerprint resolution logic that maps FingerprintProfile values to
// the corresponding utls ClientHelloID and ClientHelloSpec.

package httpclient

import (
	"fmt"

	utls "github.com/refraction-networking/utls"
)

// getNodeJSV24ClientHelloSpec returns the Node.js v24.6.0 TLS ClientHello
// specification, reproducing its exact JA3/JA4 fingerprint.
//
// Fingerprint details:
//
//	JA3:       771,4866-4867-4865-49199-49195-49200-49196-158-49191-103-49192-107-163-159-52393-52392-52394-49325-49311-49245-49249-49239-49235-162-49324-49310-49244-49248-49238-49234-49188-106-49187-64-49162-49172-57-56-49161-49171-51-50-157-49309-49233-156-49308-49232-61-60-53-47,65281-0-11-10-35-22-23-13-43-45-51,4588-29-23-30-24-25-256-257,0-1-2
//	JA3 Hash:  944d1e1858cd278718f8a46b65d3212f
//	JA4:       t13d5211_b262b3658495_8e6e362c5eac
//
// Verified against https://tls.peet.ws/ on 2025-10-17.
func getNodeJSV24ClientHelloSpec() *utls.ClientHelloSpec {
	return &utls.ClientHelloSpec{
		// TLS version range: TLS 1.3 maximum, TLS 1.2 minimum.
		TLSVersMax: utls.VersionTLS13, // 0x0304
		TLSVersMin: utls.VersionTLS12, // 0x0303

		// ===================================================================
		// Cipher suites (52) — exact order matching the real Node.js v24.6.0 JA3.
		// ===================================================================
		// JA3 Ciphers: 4866-4867-4865-49199-49195-49200-49196-158-49191-103-49192-107-163-159-52393-52392-52394-49325-49311-49245-49249-49239-49235-162-49324-49310-49244-49248-49238-49234-49188-106-49187-64-49162-49172-57-56-49161-49171-51-50-157-49309-49233-156-49308-49232-61-60-53-47
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (3)
			utls.TLS_AES_256_GCM_SHA384,       // 0x1302 = 4866
			utls.TLS_CHACHA20_POLY1305_SHA256, // 0x1303 = 4867
			utls.TLS_AES_128_GCM_SHA256,       // 0x1301 = 4865

			// TLS 1.2 ECDHE cipher suites (4, RSA first)
			utls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,   // 0xc02f = 49199
			utls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, // 0xc02b = 49195
			utls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,   // 0xc030 = 49200
			utls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, // 0xc02c = 49196

			// DHE cipher suites (7)
			0x009e, // TLS_DHE_RSA_WITH_AES_128_GCM_SHA256 = 158
			utls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, // 0xc027 = 49191
			0x0067, // TLS_DHE_RSA_WITH_AES_128_CBC_SHA256 = 103
			0xc028, // TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384 = 49192
			0x006b, // TLS_DHE_RSA_WITH_AES_256_CBC_SHA256 = 107
			0x00a3, // TLS_DHE_DSS_WITH_AES_256_GCM_SHA384 = 163
			0x009f, // TLS_DHE_RSA_WITH_AES_256_GCM_SHA384 = 159

			// ChaCha20-Poly1305 cipher suites (3)
			utls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, // 0xcca9 = 52393
			utls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,   // 0xcca8 = 52392
			0xccaa, // TLS_DHE_RSA_WITH_CHACHA20_POLY1305_SHA256 = 52394 (RFC 7905)

			// ARIA 256-bit cipher suites (6)
			0xc0ad, // TLS_ECDHE_ECDSA_WITH_ARIA_256_GCM_SHA384 = 49325
			0xc09f, // TLS_DHE_RSA_WITH_ARIA_256_GCM_SHA384 = 49311 (RFC 6209)
			0xc05d, // TLS_PSK_WITH_ARIA_256_GCM_SHA384 = 49245
			0xc061, // TLS_DHE_PSK_WITH_ARIA_256_GCM_SHA384 = 49249
			0xc057, // TLS_ECDHE_PSK_WITH_ARIA_256_CBC_SHA384 = 49239
			0xc053, // TLS_RSA_WITH_ARIA_256_GCM_SHA384 = 49235

			// DHE-DSS (1)
			0x00a2, // TLS_DHE_DSS_WITH_AES_128_GCM_SHA256 = 162

			// ARIA 128-bit cipher suites (6)
			0xc0ac, // TLS_ECDHE_ECDSA_WITH_ARIA_128_GCM_SHA256 = 49324
			0xc09e, // TLS_DHE_RSA_WITH_ARIA_128_GCM_SHA256 = 49310 (RFC 6209)
			0xc05c, // TLS_PSK_WITH_ARIA_128_GCM_SHA256 = 49244
			0xc060, // TLS_DHE_PSK_WITH_ARIA_128_GCM_SHA256 = 49248
			0xc056, // TLS_ECDHE_PSK_WITH_ARIA_128_CBC_SHA256 = 49238
			0xc052, // TLS_RSA_WITH_ARIA_128_GCM_SHA256 = 49234

			// CBC/CCM mixed cipher suites (2)
			0xc024, // TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384 = 49188 (RFC 5289)
			0x006a, // TLS_DHE_PSK_WITH_AES_256_CCM = 106

			// CBC cipher suites (9)
			0xc023, // TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256 = 49187
			0x0040, // TLS_DHE_DSS_WITH_AES_128_CBC_SHA256 = 64
			utls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, // 0xc00a = 49162
			utls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,   // 0xc014 = 49172
			0x0039,                                    // TLS_DHE_RSA_WITH_AES_256_CBC_SHA = 57
			0x0038,                                    // TLS_DHE_DSS_WITH_AES_256_CBC_SHA = 56
			utls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, // 0xc009 = 49161
			utls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,   // 0xc013 = 49171
			0x0033,                                    // TLS_DHE_RSA_WITH_AES_128_CBC_SHA = 51

			// RSA key-exchange cipher suites (10)
			utls.TLS_RSA_WITH_AES_256_GCM_SHA384, // 0x009d = 157
			0xc09d,                               // TLS_RSA_PSK_WITH_ARIA_256_GCM_SHA384 = 49309
			0xc051,                               // TLS_RSA_PSK_WITH_ARIA_256_CBC_SHA384 = 49233
			utls.TLS_RSA_WITH_AES_128_GCM_SHA256, // 0x009c = 156
			0xc09c,                               // TLS_RSA_PSK_WITH_ARIA_128_GCM_SHA256 = 49308
			0xc050,                               // TLS_RSA_PSK_WITH_ARIA_128_CBC_SHA256 = 49232
			0x003d,                               // TLS_RSA_WITH_AES_256_CBC_SHA256 = 61
			0x003c,                               // TLS_RSA_WITH_AES_128_CBC_SHA256 = 60
			utls.TLS_RSA_WITH_AES_256_CBC_SHA,    // 0x0035 = 53
			utls.TLS_RSA_WITH_AES_128_CBC_SHA,    // 0x002f = 47
		},

		// Compression methods (no compression).
		CompressionMethods: []byte{0},

		// ===================================================================
		// Extensions (11) — order is critical for fingerprint accuracy.
		// ===================================================================
		// JA3 Extensions: 65281-0-11-10-35-22-23-13-43-45-51
		Extensions: []utls.TLSExtension{
			// 1. renegotiation_info (65281) — must be first.
			&utls.RenegotiationInfoExtension{
				Renegotiation: utls.RenegotiateOnceAsClient,
			},

			// 2. server_name (0) — SNI.
			&utls.SNIExtension{},

			// 3. ec_point_formats (11).
			&utls.SupportedPointsExtension{
				SupportedPoints: []byte{0, 1, 2},
			},

			// 4. supported_groups (10).
			&utls.SupportedCurvesExtension{
				Curves: []utls.CurveID{
					0x11ec,         // X25519MLKEM768 = 4588
					utls.X25519,    // 29
					utls.CurveP256, // 23
					0x001e,         // X448 = 30
					utls.CurveP384, // 24
					utls.CurveP521, // 25
					0x0100,         // ffdhe2048 = 256
					0x0101,         // ffdhe3072 = 257
				},
			},

			// 5. session_ticket (35).
			&utls.SessionTicketExtension{},

			// 6. encrypt_then_mac (22).
			&utls.GenericExtension{
				Id:   22,
				Data: []byte{},
			},

			// 7. extended_master_secret (23).
			&utls.ExtendedMasterSecretExtension{},

			// 8. signature_algorithms (13).
			&utls.SignatureAlgorithmsExtension{
				SupportedSignatureAlgorithms: []utls.SignatureScheme{
					0x0905, 0x0906, 0x0904, // custom signature algorithms
					utls.ECDSAWithP256AndSHA256, // 0x0403
					utls.ECDSAWithP384AndSHA384, // 0x0503
					utls.ECDSAWithP521AndSHA512, // 0x0603
					utls.Ed25519, utls.Ed25519,  // 0x0807 (appears twice)
					0x081a, 0x081b, 0x081c, // BrainpoolP256r1tls13_sha256, BrainpoolP384r1tls13_sha384, BrainpoolP512r1tls13_sha512
					0x0809, 0x080a, 0x080b, // rsa_pss_pss_sha256, rsa_pss_pss_sha384, rsa_pss_pss_sha512
					utls.PSSWithSHA256, utls.PSSWithSHA384, utls.PSSWithSHA512, // 0x0804, 0x0805, 0x0806 (rsa_pss_rsae)
					utls.PKCS1WithSHA256, utls.PKCS1WithSHA384, utls.PKCS1WithSHA512, // 0x0401, 0x0501, 0x0601
					0x0303, 0x0301, 0x0302, // SHA224 variants
					0x0402, 0x0502, 0x0602, // DSA variants
				},
			},

			// 9. supported_versions (43).
			&utls.SupportedVersionsExtension{
				Versions: []uint16{
					utls.VersionTLS13,
					utls.VersionTLS12,
				},
			},

			// 10. psk_key_exchange_modes (45).
			&utls.PSKKeyExchangeModesExtension{
				Modes: []uint8{1},
			},

			// 11. key_share (51).
			&utls.KeyShareExtension{
				KeyShares: []utls.KeyShare{
					{Group: 0x11ec},
					{Group: utls.X25519},
				},
			},
		},

		// Session ID handling — leave as default.
		GetSessionID: nil,
	}
}

// resolveFingerprint maps a FingerprintProfile to its corresponding utls
// ClientHelloID and optional ClientHelloSpec.
//
// For FingerprintNodeJSV24 (and the empty string) it returns the built-in
// Node.js v24.6.0 spec together with utls.HelloCustom. For the preset browser
// profiles (Chrome, Firefox, Randomized) it returns the matching utls
// ClientHelloID with a nil spec, letting utls manage the fingerprint
// internally. For FingerprintCustom it requires a non-nil customSpec and
// returns it with utls.HelloCustom.
//
// Returns an error for FingerprintCustom when customSpec is nil, or for any
// unrecognized profile value.
func resolveFingerprint(profile FingerprintProfile, customSpec *utls.ClientHelloSpec) (utls.ClientHelloID, *utls.ClientHelloSpec, error) {
	switch profile {
	case FingerprintNodeJSV24, "":
		return utls.HelloCustom, getNodeJSV24ClientHelloSpec(), nil

	case FingerprintChromeAuto:
		return utls.HelloChrome_Auto, nil, nil

	case FingerprintFirefoxAuto:
		return utls.HelloFirefox_Auto, nil, nil

	case FingerprintRandomized:
		return utls.HelloRandomized, nil, nil

	case FingerprintCustom:
		if customSpec == nil {
			return utls.ClientHelloID{}, nil, fmt.Errorf(`custom fingerprint profile requires a non-nil ClientHelloSpec`)
		}
		return utls.HelloCustom, customSpec, nil

	default:
		return utls.ClientHelloID{}, nil, fmt.Errorf(`unsupported fingerprint profile: %s`, profile)
	}
}
