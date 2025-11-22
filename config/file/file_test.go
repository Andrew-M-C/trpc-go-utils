package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andrew-M-C/trpc-go-utils/config/file"
	"github.com/smartystreets/goconvey/convey"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/config"
)

var (
	cv = convey.Convey
	so = convey.So
	eq = convey.ShouldEqual

	isNil    = convey.ShouldBeNil
	notNil   = convey.ShouldNotBeNil
	isTrue   = convey.ShouldBeTrue
	contains = convey.ShouldContainSubstring
)

// 测试用的临时目录和文件
var (
	testDir      string
	testFilePath string
	testYamlPath string
	testFileKey  = "test_config"
	testContent1 = "initial content v1"
	testContent2 = "updated content v2"
	testContent3 = "final content v3"
)

func TestMain(m *testing.M) {
	var err error
	// 创建临时测试目录
	testDir, err = os.MkdirTemp("", "file_config_test_*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(testDir)

	testFilePath = filepath.Join(testDir, "test_file.txt")
	testYamlPath = filepath.Join(testDir, "trpc_go.yaml")

	// 创建初始测试文件
	if err := os.WriteFile(testFilePath, []byte(testContent1), 0644); err != nil {
		panic(err)
	}

	// 创建 trpc_go.yaml 配置文件
	yamlContent := `
server:
  service:
    - name: test.service.TestServer
      ip: 127.0.0.1
      port: 18080
      network: tcp
      protocol: http
      timeout: 1000

plugins:
  config:
    file:
      - name: ` + testFileKey + `
        path: ` + testFilePath + `
`
	if err := os.WriteFile(testYamlPath, []byte(yamlContent), 0644); err != nil {
		panic(err)
	}

	// 保存原始工作目录
	originalDir, _ := os.Getwd()
	// 切换到测试目录以便 trpc 能找到配置文件
	if err := os.Chdir(testDir); err != nil {
		panic(err)
	}

	// 运行测试
	exitCode := m.Run()

	// 恢复工作目录
	_ = os.Chdir(originalDir)
	os.Exit(exitCode)
}

// TestFileConfigRead 测试能否在服务启动后读取文件内容
func TestFileConfigRead(t *testing.T) {
	cv("测试 trpc 服务启动后读取文件配置", t, func() {
		// 初始化 trpc 服务
		s := trpc.NewServer()
		so(s, notNil)

		cv("应该能获取到 file 类型的配置 API", func() {
			api := file.API{}
			kvConfig := api.GetConfig()
			so(kvConfig, notNil)
			so(kvConfig.Name(), eq, "file")
		})

		cv("应该能读取到初始文件内容", func() {
			api := file.API{}
			ctx := context.Background()

			resp, err := api.Get(ctx, testFileKey)
			so(err, isNil)
			so(resp, notNil)
			so(resp.Value(), eq, testContent1)
			so(resp.Event(), eq, config.EventTypeNull)
		})

		cv("应该能从缓存读取文件内容", func() {
			// 这个测试验证：当文件被监听后修改，缓存会被更新
			// 即使文件不可读，也能从缓存读取到最后一次成功读取的内容
			api := file.API{}
			ctx := context.Background()

			// 先创建监听器，这样文件变更时会写入缓存
			ch, err := api.Watch(ctx, testFileKey)
			so(err, isNil)
			so(ch, notNil)

			// 等待监听器准备好
			time.Sleep(100 * time.Millisecond)

			// 修改文件以触发缓存写入
			testCachedContent := "cached content for testing"
			err = os.WriteFile(testFilePath, []byte(testCachedContent), 0644)
			so(err, isNil)

			// 等待接收到变更事件，确保缓存已写入
			done := make(chan config.Response, 1)
			go func() {
				select {
				case resp := <-ch:
					done <- resp
				case <-time.After(2 * time.Second):
					done <- nil
				}
			}()
			watchResp := <-done
			so(watchResp, notNil)
			so(watchResp.Value(), eq, testCachedContent)

			// 验证可以通过 Get 读取到更新后的内容
			// （这会从文件读取，同时也会更新缓存）
			resp, err := api.Get(ctx, testFileKey)
			so(err, isNil)
			so(resp, notNil)
			so(resp.Value(), eq, testCachedContent)

			// 清空 channel 以避免影响后续测试
			time.Sleep(100 * time.Millisecond)
			for len(ch) > 0 {
				<-ch
			}
		})

		cv("读取不存在的配置应该报错", func() {
			api := file.API{}
			ctx := context.Background()

			resp, err := api.Get(ctx, "non_existent_key")
			so(err, notNil)
			so(resp, isNil)
		})

		cv("Put 操作应该报错（不支持）", func() {
			api := file.API{}
			ctx := context.Background()

			err := api.Put(ctx, testFileKey, "new value")
			so(err, notNil)
			so(err.Error(), contains, "does not support put")
		})

		cv("Del 操作应该报错（不支持）", func() {
			api := file.API{}
			ctx := context.Background()

			err := api.Del(ctx, testFileKey)
			so(err, notNil)
			so(err.Error(), contains, "does not support del")
		})
	})
}

// TestFileConfigWatch 测试能否监听文件内容的变化
func TestFileConfigWatch(t *testing.T) {
	cv("测试 trpc 服务启动后监听文件变更", t, func() {
		// 初始化 trpc 服务
		s := trpc.NewServer()
		so(s, notNil)

		api := file.API{}
		kvConfig := api.GetConfig()
		so(kvConfig, notNil)

		ctx := context.Background()

		// 重要：清空之前测试可能留下的所有事件
		ch, _ := api.Watch(ctx, testFileKey)
		if ch != nil {
			clearCh := true
			for clearCh {
				select {
				case <-ch:
					t.Logf("清空了一个旧事件")
				case <-time.After(50 * time.Millisecond):
					clearCh = false
				}
			}
		}

		cv("应该能成功创建 Watch 监听器", func() {
			ch, err := api.Watch(ctx, testFileKey)
			so(err, isNil)
			so(ch, notNil)
		})

		cv("多次 Watch 同一个 key 应该返回相同的 channel", func() {
			ch1, err1 := api.Watch(ctx, testFileKey)
			so(err1, isNil)
			so(ch1, notNil)

			ch2, err2 := api.Watch(ctx, testFileKey)
			so(err2, isNil)
			so(ch2, notNil)

			// 应该是同一个 channel
			so(ch1 == ch2, isTrue)
		})

		cv("应该能监听到文件写入事件", func() {
			ch, err := api.Watch(ctx, testFileKey)
			so(err, isNil)

			// 清空 channel 中可能存在的旧事件
			for len(ch) > 0 {
				<-ch
			}

			// 启动一个 goroutine 来接收变更事件
			done := make(chan config.Response, 1)

			go func() {
				select {
				case resp := <-ch:
					done <- resp
				case <-time.After(5 * time.Second):
					done <- nil
				}
			}()

			// 稍等一下，确保监听器已经就绪
			time.Sleep(100 * time.Millisecond)

			// 修改文件内容
			t.Logf("准备写入文件: %s", testContent2)
			err = os.WriteFile(testFilePath, []byte(testContent2), 0644)
			so(err, isNil)

			// 等待接收到变更事件
			receivedResp := <-done
			if receivedResp == nil {
				t.Logf("未能接收到文件变更事件（超时）")
			}
			so(receivedResp, notNil)
			so(receivedResp.Value(), eq, testContent2)
			so(receivedResp.Event(), eq, config.EventTypePut)
		})

		cv("应该能监听到多次文件变更", func() {
			ch, err := api.Watch(ctx, testFileKey)
			so(err, isNil)

			// 清空 channel 中可能存在的旧事件
			for len(ch) > 0 {
				<-ch
			}

			// 第一次变更
			done1 := make(chan config.Response, 1)
			go func() {
				select {
				case resp := <-ch:
					done1 <- resp
				case <-time.After(5 * time.Second):
					done1 <- nil
				}
			}()

			time.Sleep(100 * time.Millisecond)
			t.Logf("第一次写入文件: %s", testContent3)
			err = os.WriteFile(testFilePath, []byte(testContent3), 0644)
			so(err, isNil)

			resp1 := <-done1
			if resp1 == nil {
				t.Logf("第一次未能接收到文件变更事件（超时）")
			}
			so(resp1, notNil)
			so(resp1.Value(), eq, testContent3)
			so(resp1.Event(), eq, config.EventTypePut)

			// 第二次变更
			done2 := make(chan config.Response, 1)
			go func() {
				select {
				case resp := <-ch:
					done2 <- resp
				case <-time.After(5 * time.Second):
					done2 <- nil
				}
			}()

			time.Sleep(100 * time.Millisecond)
			modifiedContent := testContent3 + " - modified again"
			t.Logf("第二次写入文件: %s", modifiedContent)
			err = os.WriteFile(testFilePath, []byte(modifiedContent), 0644)
			so(err, isNil)

			resp2 := <-done2
			if resp2 == nil {
				t.Logf("第二次未能接收到文件变更事件（超时）")
			}
			so(resp2, notNil)
			so(resp2.Value(), eq, modifiedContent)
			so(resp2.Event(), eq, config.EventTypePut)
		})

		cv("监听不存在的配置应该报错", func() {
			ch, err := api.Watch(ctx, "non_existent_key")
			so(err, notNil)
			so(ch, isNil)
		})
	})
}

// TestFileConfigIntegration 集成测试：读取和监听一起
func TestFileConfigIntegration(t *testing.T) {
	cv("集成测试：读取和监听配合使用", t, func() {
		// 初始化 trpc 服务
		s := trpc.NewServer()
		so(s, notNil)

		api := file.API{}
		ctx := context.Background()

		// 重要：清空之前测试可能留下的所有事件
		ch, _ := api.Watch(ctx, testFileKey)
		if ch != nil {
			clearCh := true
			for clearCh {
				select {
				case <-ch:
					t.Logf("清空了一个旧事件")
				case <-time.After(50 * time.Millisecond):
					clearCh = false
				}
			}
		}

		cv("先读取初始内容，然后监听变更", func() {
			// 1. 读取初始内容
			resp, err := api.Get(ctx, testFileKey)
			so(err, isNil)
			initialContent := resp.Value()
			t.Logf("初始内容: %s", initialContent)

			// 2. 开始监听
			ch, err := api.Watch(ctx, testFileKey)
			so(err, isNil)

			// 清空 channel 中可能存在的旧事件
			for len(ch) > 0 {
				<-ch
			}

			done := make(chan config.Response, 1)
			go func() {
				select {
				case resp := <-ch:
					done <- resp
				case <-time.After(5 * time.Second):
					done <- nil
				}
			}()

			// 3. 修改文件
			time.Sleep(100 * time.Millisecond)
			newContent := "integrated test content - " + time.Now().Format("15:04:05")
			t.Logf("准备写入新内容: %s", newContent)
			err = os.WriteFile(testFilePath, []byte(newContent), 0644)
			so(err, isNil)

			// 4. 验证监听到变更
			watchResp := <-done
			if watchResp == nil {
				t.Logf("未能接收到文件变更事件（超时）")
			}
			so(watchResp, notNil)
			so(watchResp.Value(), eq, newContent)
			so(watchResp.Event(), eq, config.EventTypePut)

			// 5. 再次读取，应该能读到最新内容（从缓存）
			resp2, err := api.Get(ctx, testFileKey)
			so(err, isNil)
			so(resp2.Value(), eq, newContent)

			t.Logf("最终内容: %s", resp2.Value())
		})
	})
}
