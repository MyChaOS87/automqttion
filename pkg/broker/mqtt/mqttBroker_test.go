package mqtt_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt"
	"github.com/MyChaOS87/automqttion/pkg/broker/mqtt/mqtttest"
)

var errSimBrokerFailure = errors.New("simulated broker failure")

type samplePayload struct {
	Src int `json:"Src"`
	Cmd int `json:"Cmd"`
}

type subscribeTestCases []struct {
	name             string
	useAnyType       bool
	subErr           error
	inboundPayload   string
	expectDelivered  bool
	expectedReceived any
	expectedAsyncErr *mqtt.Error
}

func TestBrokerSubscribe(t *testing.T) {
	t.Parallel()

	tests := subscribeTestCases{
		{
			name:             "typed happy path unmarshals into struct",
			useAnyType:       false,
			inboundPayload:   `{"Src":34,"Cmd":104}`,
			expectDelivered:  true,
			expectedReceived: samplePayload{Src: 34, Cmd: 104},
		},
		{
			name:             "untyped happy path unmarshals into map",
			useAnyType:       true,
			inboundPayload:   `{"Src":34,"Payload":"MAQ="}`,
			expectDelivered:  true,
			expectedReceived: map[string]any{"Src": float64(34), "Payload": "MAQ="},
		},
		{
			name:            "subscribe token error reports OpSubscribe without delivering messages",
			useAnyType:      false,
			subErr:          errSimBrokerFailure,
			inboundPayload:  `{"Src":34,"Cmd":104}`,
			expectDelivered: false,
			expectedAsyncErr: &mqtt.Error{
				Op:    mqtt.OpSubscribe,
				Topic: "lcn/segment/0/target/4/",
				Err:   errSimBrokerFailure,
			},
		},
		{
			name:            "invalid JSON payload reports OpUnmarshal and skips callback",
			useAnyType:      false,
			inboundPayload:  `{invalid-json`,
			expectDelivered: true,
			expectedAsyncErr: &mqtt.Error{
				Op:      mqtt.OpUnmarshal,
				Topic:   "lcn/segment/0/target/4/",
				Payload: []byte(`{invalid-json`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				errCh := make(chan *mqtt.Error, 1)

				brk, sim := mqtttest.NewBroker(
					mqtt.BrokerURL("tcp://127.0.0.1:1883"),
					mqtt.TLS(&tls.Config{MinVersion: tls.VersionTLS12}),
					mqtt.OnError(func(err *mqtt.Error) {
						errCh <- err
					}),
				)
				sim.SubErr = tt.subErr

				require.NotNil(t, brk)

				runErrCh := make(chan error, 1)

				go func() {
					runErrCh <- brk.Run(ctx)
				}()

				synctest.Wait()

				var received any

				topic := brk.Topic("lcn/segment/0/target/4/")
				if tt.useAnyType {
					topic.Subscribe(func(_ string, data any) {
						received = data
					})
				} else {
					topic.Subscribe(func(_ string, data samplePayload) {
						received = data
					})
				}

				synctest.Wait()

				assert.Equal(t, []string{"lcn/segment/0/target/4/"}, sim.SubscribedTopics())
				assert.Equal(t, tt.expectDelivered, sim.Deliver("lcn/segment/0/target/4/", tt.inboundPayload))
				assert.Equal(t, tt.expectedReceived, received)

				if tt.expectedAsyncErr != nil {
					got := <-errCh
					assert.Equal(t, tt.expectedAsyncErr.Op, got.Op)
					assert.Equal(t, tt.expectedAsyncErr.Topic, got.Topic)
					assert.Equal(t, tt.expectedAsyncErr.Payload, got.Payload)
					require.Error(t, got.Unwrap())
					assert.Contains(t, got.Error(), string(tt.expectedAsyncErr.Op))

					if tt.expectedAsyncErr.Err != nil {
						require.ErrorIs(t, got, tt.expectedAsyncErr.Err)
					}
				} else {
					assert.Empty(t, errCh)
				}

				cancel()
				require.NoError(t, <-runErrCh)
				assert.True(t, sim.Disconnected())
			})
		})
	}
}

type reconnectScenarioTestCases []struct {
	name             string
	scenario         func(t *testing.T, sim *mqtttest.Simulator, registerSub func(), run func())
	expectedSubCalls []string
}

func TestBrokerReconnect(t *testing.T) {
	t.Parallel()

	const targetTopic = "lcn/segment/0/target/4/"

	tests := reconnectScenarioTestCases{
		{
			name: "subscribe before Run subscribes on connect and re-subscribes across two reconnects",
			scenario: func(t *testing.T, sim *mqtttest.Simulator, registerSub func(), run func()) {
				t.Helper()

				registerSub()
				assert.Empty(t, sim.SubscribedTopics())

				run()
				synctest.Wait()

				sim.SimulateReconnect(errSimBrokerFailure)
				synctest.Wait()

				sim.SimulateReconnect(errSimBrokerFailure)
				synctest.Wait()
			},
			expectedSubCalls: []string{targetTopic, targetTopic, targetTopic},
		},
		{
			name: "subscribe after Run connected re-subscribes across reconnect",
			scenario: func(t *testing.T, sim *mqtttest.Simulator, registerSub func(), run func()) {
				t.Helper()

				run()
				synctest.Wait()

				registerSub()
				synctest.Wait()

				sim.SimulateReconnect(errSimBrokerFailure)
				synctest.Wait()
			},
			expectedSubCalls: []string{targetTopic, targetTopic},
		},
		{
			name: "subscribe during startup before OnConnect executes subscribes exactly once",
			scenario: func(t *testing.T, sim *mqtttest.Simulator, registerSub func(), run func()) {
				t.Helper()

				sim.DeferOnConnect = true

				run()
				synctest.Wait()

				registerSub()
				synctest.Wait()
				assert.Empty(t, sim.SubscribedTopics())

				sim.SimulateConnect()
				synctest.Wait()
			},
			expectedSubCalls: []string{targetTopic},
		},
		{
			name: "subscribe while disconnected defers client subscribe until SimulateConnect",
			scenario: func(t *testing.T, sim *mqtttest.Simulator, registerSub func(), run func()) {
				t.Helper()

				run()
				synctest.Wait()

				sim.SimulateDisconnect(errSimBrokerFailure)
				synctest.Wait()

				registerSub()
				synctest.Wait()
				assert.Empty(t, sim.SubscribedTopics())
				assert.False(t, sim.Deliver(targetTopic, `{"Src":34,"Cmd":104}`))

				sim.SimulateConnect()
				synctest.Wait()
			},
			expectedSubCalls: []string{targetTopic},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				errCh := make(chan *mqtt.Error, 1)

				brk, sim := mqtttest.NewBroker(
					mqtt.OnError(func(err *mqtt.Error) {
						errCh <- err
					}),
				)

				var received samplePayload

				registerSub := func() {
					brk.Topic(targetTopic).Subscribe(func(_ string, data samplePayload) {
						received = data
					})
				}

				runErrCh := make(chan error, 1)
				run := func() {
					go func() {
						runErrCh <- brk.Run(ctx)
					}()
				}

				tt.scenario(t, sim, registerSub, run)

				assert.Equal(t, tt.expectedSubCalls, sim.SubscribedTopics())
				require.True(t, sim.Deliver(targetTopic, `{"Src":34,"Cmd":104}`))
				assert.Equal(t, samplePayload{Src: 34, Cmd: 104}, received)
				assert.Empty(t, errCh)

				cancel()
				require.NoError(t, <-runErrCh)
				assert.True(t, sim.Disconnected())
			})
		})
	}
}

type publishTestCases []struct {
	name             string
	topic            string
	isString         bool
	payload          any
	pubErr           error
	expectSyncErr    bool
	expectedRecords  []mqtttest.PublishedMessage
	expectedAsyncErr *mqtt.Error
}

func TestBrokerPublish(t *testing.T) {
	t.Parallel()

	tests := publishTestCases{
		{
			name:     "PublishString sends plain payload with QoS 1",
			topic:    "device/switch/command:0",
			isString: true,
			payload:  "on",
			expectedRecords: []mqtttest.PublishedMessage{
				{
					Topic:   "device/switch/command:0",
					QoS:     1,
					Retain:  false,
					Payload: "on",
				},
			},
		},
		{
			name:     "Publish marshals struct to JSON with QoS 1",
			topic:    "lcn/in",
			isString: false,
			payload:  samplePayload{Src: 34, Cmd: 104},
			expectedRecords: []mqtttest.PublishedMessage{
				{
					Topic:   "lcn/in",
					QoS:     1,
					Retain:  false,
					Payload: `{"Src":34,"Cmd":104}`,
				},
			},
		},
		{
			name:            "Publish with unmarshalable value returns synchronous marshal error and skips client publish",
			topic:           "lcn/in",
			isString:        false,
			payload:         make(chan int),
			expectSyncErr:   true,
			expectedRecords: nil,
		},
		{
			name:     "Publish token error reports OpPublish with topic and payload",
			topic:    "lcn/in",
			isString: true,
			payload:  "off",
			pubErr:   errSimBrokerFailure,
			expectedRecords: []mqtttest.PublishedMessage{
				{
					Topic:   "lcn/in",
					QoS:     1,
					Retain:  false,
					Payload: "off",
				},
			},
			expectedAsyncErr: &mqtt.Error{
				Op:      mqtt.OpPublish,
				Topic:   "lcn/in",
				Payload: []byte("off"),
				Err:     errSimBrokerFailure,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				errCh := make(chan *mqtt.Error, 1)

				brk, sim := mqtttest.NewBroker(
					mqtt.OnError(func(err *mqtt.Error) {
						errCh <- err
					}),
				)
				sim.PubErr = tt.pubErr

				topic := brk.Topic(tt.topic)
				if tt.isString {
					s, ok := tt.payload.(string)
					require.True(t, ok)
					topic.PublishString(s)
				} else {
					err := topic.Publish(tt.payload)
					if tt.expectSyncErr {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}
				}

				synctest.Wait()

				assert.Equal(t, tt.expectedRecords, sim.PublishedMessages())

				if tt.expectedAsyncErr != nil {
					got := <-errCh
					assert.Equal(t, tt.expectedAsyncErr.Op, got.Op)
					assert.Equal(t, tt.expectedAsyncErr.Topic, got.Topic)
					assert.Equal(t, tt.expectedAsyncErr.Payload, got.Payload)
					require.ErrorIs(t, got, tt.expectedAsyncErr.Err)
				} else {
					assert.Empty(t, errCh)
				}
			})
		})
	}
}

type runLifecycleTestCases []struct {
	name               string
	connectErr         error
	blockConnect       bool
	cancelBeforeRun    bool
	expectedErr        error
	expectDisconnected bool
}

func TestBrokerRunLifecycle(t *testing.T) {
	t.Parallel()

	tests := runLifecycleTestCases{
		{
			name:               "connect error returns wrapped error",
			connectErr:         errSimBrokerFailure,
			expectedErr:        errSimBrokerFailure,
			expectDisconnected: false,
		},
		{
			name:               "cancelled context during connect disconnects client and returns context error",
			blockConnect:       true,
			cancelBeforeRun:    true,
			expectedErr:        context.Canceled,
			expectDisconnected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			if tt.cancelBeforeRun {
				cancel()
			}

			brk, sim := mqtttest.NewBroker()
			sim.ConnectErr = tt.connectErr
			sim.BlockConnect = tt.blockConnect

			err := brk.Run(ctx)
			require.ErrorIs(t, err, tt.expectedErr)
			assert.Equal(t, tt.expectDisconnected, sim.Disconnected())
		})
	}
}

func TestBrokerOptionsAndTokenTimeouts(t *testing.T) {
	t.Parallel()

	t.Run("wires ClientID, Credentials, and graceful 250ms disconnect quiesce", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			const expectedQuiesce uint = 250

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			brk, sim := mqtttest.NewBroker(
				mqtt.BrokerURL("tls://mosquitto.internal:8883"),
				mqtt.ClientID("lcn2mqtt-test"),
				mqtt.Credentials("bridge-user", "bridge-pass"),
			)

			opts := sim.ClientOptions()
			require.NotNil(t, opts)
			assert.Equal(t, "lcn2mqtt-test", opts.ClientID)
			assert.Equal(t, "bridge-user", opts.Username)
			assert.Equal(t, "bridge-pass", opts.Password)

			runErr := make(chan error, 1)

			go func() {
				runErr <- brk.Run(ctx)
			}()

			synctest.Wait()
			cancel()
			require.NoError(t, <-runErr)
			assert.Equal(t, expectedQuiesce, sim.DisconnectQuiesce())
		})
	})

	t.Run("stalled publish token times out with ErrTokenTimeout and DeadlineExceeded", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			const timeout = 50 * time.Millisecond

			errCh := make(chan *mqtt.Error, 1)

			brk, sim := mqtttest.NewBroker(
				mqtt.TokenTimeout(timeout),
				mqtt.OnError(func(err *mqtt.Error) {
					errCh <- err
				}),
			)
			sim.BlockPub = true

			brk.Topic("lcn/in").PublishString("payload")

			time.Sleep(timeout)
			synctest.Wait()

			got := <-errCh
			assert.Equal(t, mqtt.OpPublish, got.Op)
			assert.Equal(t, "lcn/in", got.Topic)
			assert.Equal(t, []byte("payload"), got.Payload)
			require.ErrorIs(t, got, mqtt.ErrTokenTimeout)
			require.ErrorIs(t, got, context.DeadlineExceeded)
		})
	})

	t.Run("stalled subscribe token times out with ErrTokenTimeout and DeadlineExceeded", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			const timeout = 50 * time.Millisecond

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			errCh := make(chan *mqtt.Error, 1)

			brk, sim := mqtttest.NewBroker(
				mqtt.TokenTimeout(timeout),
				mqtt.OnError(func(err *mqtt.Error) {
					errCh <- err
				}),
			)
			sim.BlockSub = true

			brk.Topic("lcn/in").Subscribe(func(_ string, _ samplePayload) {})

			runErr := make(chan error, 1)

			go func() {
				runErr <- brk.Run(ctx)
			}()

			synctest.Wait()
			time.Sleep(timeout)
			synctest.Wait()

			got := <-errCh
			assert.Equal(t, mqtt.OpSubscribe, got.Op)
			assert.Equal(t, "lcn/in", got.Topic)
			require.ErrorIs(t, got, mqtt.ErrTokenTimeout)
			require.ErrorIs(t, got, context.DeadlineExceeded)

			cancel()
			require.NoError(t, <-runErr)
		})
	})
}

type subscribeRawTestCases []struct {
	name             string
	inboundPayloads  []string
	expectedPayloads [][]byte
}

func TestBrokerSubscribeRaw(t *testing.T) {
	t.Parallel()

	tests := subscribeRawTestCases{
		{
			name:            "delivers plain-text and arbitrary raw payloads without JSON unmarshal",
			inboundPayloads: []string{"on", "off", "online", `{"raw":true}`, "\x00\xff"},
			expectedPayloads: [][]byte{
				[]byte("on"),
				[]byte("off"),
				[]byte("online"),
				[]byte(`{"raw":true}`),
				[]byte("\x00\xff"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				errCh := make(chan *mqtt.Error, 1)

				brk, sim := mqtttest.NewBroker(
					mqtt.BrokerURL("tcp://127.0.0.1:1883"),
					mqtt.ClientID("mqtt-raw-test"),
					mqtt.Credentials("user1", "pass1"),
					mqtt.OnError(func(err *mqtt.Error) {
						errCh <- err
					}),
				)

				require.Equal(t, "mqtt-raw-test", sim.ClientOptions().ClientID)
				require.Equal(t, "user1", sim.ClientOptions().Username)
				require.Equal(t, "pass1", sim.ClientOptions().Password)

				var received [][]byte

				brk.Topic("shellyplug/status/switch:0").SubscribeRaw(func(_ string, data []byte) {
					received = append(received, data)
				})

				runErrCh := make(chan error, 1)

				go func() {
					runErrCh <- brk.Run(ctx)
				}()

				synctest.Wait()

				for _, p := range tt.inboundPayloads {
					require.True(t, sim.Deliver("shellyplug/status/switch:0", p))
				}

				assert.Equal(t, tt.expectedPayloads, received)
				assert.Empty(t, errCh)

				cancel()
				require.NoError(t, <-runErrCh)
			})
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	caPath, certPath, keyPath := writeTestPKI(t, dir)
	badPEMPath := filepath.Join(dir, "bad-ca.pem")
	require.NoError(t, os.WriteFile(badPEMPath, []byte("not-a-pem"), 0o600))

	tests := []struct {
		name       string
		cfg        mqtt.TLSConfig
		wantErr    error
		wantRoots  bool
		wantCerts  int
		wantServer string
	}{
		{
			name: "default TLS config enforces TLS 1.2+",
			cfg: mqtt.TLSConfig{
				Enabled:    true,
				ServerName: "mosquitto.internal",
			},
			wantServer: "mosquitto.internal",
		},
		{
			name: "loads custom CA and mutual-TLS client certificate pair",
			cfg: mqtt.TLSConfig{
				Enabled:  true,
				CAFile:   caPath,
				CertFile: certPath,
				KeyFile:  keyPath,
			},
			wantRoots: true,
			wantCerts: 1,
		},
		{
			name: "certFile without keyFile returns ErrInvalidTLSConfig",
			cfg: mqtt.TLSConfig{
				Enabled:  true,
				CertFile: certPath,
			},
			wantErr: mqtt.ErrInvalidTLSConfig,
		},
		{
			name: "missing CAFile returns ErrInvalidTLSConfig",
			cfg: mqtt.TLSConfig{
				Enabled: true,
				CAFile:  filepath.Join(dir, "missing.pem"),
			},
			wantErr: mqtt.ErrInvalidTLSConfig,
		},
		{
			name: "malformed CA PEM returns ErrInvalidTLSConfig",
			cfg: mqtt.TLSConfig{
				Enabled: true,
				CAFile:  badPEMPath,
			},
			wantErr: mqtt.ErrInvalidTLSConfig,
		},
		{
			name: "mismatched key file returns ErrInvalidTLSConfig",
			cfg: mqtt.TLSConfig{
				Enabled:  true,
				CertFile: certPath,
				KeyFile:  badPEMPath,
			},
			wantErr: mqtt.ErrInvalidTLSConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tlsCfg, err := mqtt.BuildTLSConfig(tt.cfg)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, tlsCfg)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, tlsCfg)
			assert.Equal(t, uint16(tls.VersionTLS12), tlsCfg.MinVersion)
			assert.Equal(t, tt.wantServer, tlsCfg.ServerName)
			assert.Equal(t, tt.wantRoots, tlsCfg.RootCAs != nil)
			assert.Len(t, tlsCfg.Certificates, tt.wantCerts)
		})
	}
}

func writeTestPKI(t *testing.T, dir string) (caPath, certPath, keyPath string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mqtt-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	caPath = filepath.Join(dir, "ca.crt")
	certPath = filepath.Join(dir, "client.crt")
	keyPath = filepath.Join(dir, "client.key")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	require.NoError(t, os.WriteFile(caPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(certPath, certPEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0o600))

	return caPath, certPath, keyPath
}
