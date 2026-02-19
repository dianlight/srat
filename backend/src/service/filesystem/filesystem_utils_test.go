package filesystem_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/service/filesystem"
)

func TestFilesystemUtilsWithoutMockedCommands(t *testing.T) {
	registry := filesystem.NewRegistry()
	adapters := registry.GetAll()

	if len(adapters) == 0 {
		t.Fatal("expected at least one filesystem adapter in registry")
	}

	for _, adapter := range adapters {
		adapter := adapter
		t.Run(adapter.GetName(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			support, supportErr := adapter.IsSupported(ctx)
			if supportErr != nil {
				t.Fatalf("IsSupported failed for %s: %v", adapter.GetName(), supportErr)
			}

			// check if exists a test device image in backend/test and use it instead of creating a new one, to speed up tests and reduce resource usage and avoid issues with some filesystems that don't work well with sparse files (like hfsplus)
			deviceFile := fmt.Sprintf("test/%s_test_device.img", adapter.GetName())
			if _, err := os.Stat(deviceFile); err == nil {
				t.Logf("using existing test device image for %s: %s", adapter.GetName(), deviceFile)
			} else {
				t.Logf("creating new test device image for %s", adapter.GetName())
				testDevicePath, supportErr := createSparseDeviceFile(t, adapter.GetName())
				if supportErr != nil {
					t.Fatalf("failed to create test device file for %s: %v", adapter.GetName(), supportErr)
				}
				deviceFile = testDevicePath
			}

			label := testLabelFor(adapter.GetName())

			t.Run(adapter.GetName()+"_format", func(t *testing.T) {
				if support.CanFormat {
					formatErr := adapter.Format(ctx, deviceFile, dto.FormatOptions{
						Label: label,
						Force: true,
					}, nil)
					if formatErr != nil {
						t.Fatalf("format attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, formatErr)
					}
				} else {
					t.Skipf("skipping format test for %s as it is not supported", adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_state", func(t *testing.T) {
				if support.CanGetState {
					_, stateErr := adapter.GetState(ctx, deviceFile)
					if stateErr != nil {
						t.Logf("state attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, stateErr)
					}
				} else {
					t.Skipf("skipping state test for %s as it is not supported", adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_check", func(t *testing.T) {
				if support.CanCheck {
					_, checkErr := adapter.Check(ctx, deviceFile, dto.CheckOptions{
						AutoFix: false,
						Force:   true,
						Verbose: false,
					}, nil)
					if checkErr != nil {
						t.Logf("check attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, checkErr)
					}
				} else {
					t.Skipf("skipping check test for %s as it is not supported", adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_label", func(t *testing.T) {
				if support.CanSetLabel {
					setLabelErr := adapter.SetLabel(ctx, deviceFile, label)
					if setLabelErr != nil {
						t.Logf("set label attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, setLabelErr)
					}
					getLabel, getLabelErr := adapter.GetLabel(ctx, deviceFile)
					if getLabelErr != nil {
						t.Errorf("get label attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, getLabelErr)
					} else if getLabel != label {
						t.Errorf("label mismatch for %s on %s: expected '%s', got '%s'", adapter.GetName(), deviceFile, label, getLabel)
					}
				} else {
					t.Skipf("skipping set label test for %s as it is not supported", adapter.GetName())
				}
			})

			/*

				// mount test
				t.Run(adapter.GetName()+"_mount", func(t *testing.T) {
					if support.CanMount {
						mountPoint, err := os.MkdirTemp("", fmt.Sprintf("%s-mount-*", adapter.GetName()))
						if err != nil {
							t.Fatalf("failed to create mount point for %s: %v", adapter.GetName(), err)
						}
						t.Cleanup(func() {
							_ = os.RemoveAll(mountPoint)
						})

						mountErr := adapter.Mount(ctx, deviceFile, mountPoint, nil)
						if mountErr != nil {
							t.Logf("mount attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, mountErr)
							return
						}

						// check if mount point is listed in mount command output
						mountOutput, _, _ := runCommandCached(ctx, "mount")
						if !strings.Contains(mountOutput, mountPoint) {
							t.Errorf("mount point not found in mount output for %s on %s: %s", adapter.GetName(), deviceFile, mountOutput)
						}

						// unmount after test
						unmountErr := adapter.Unmount(ctx, mountPoint)
						if unmountErr != nil {
							t.Logf("unmount attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, unmountErr)
						}
					} else {
						t.Skipf("skipping mount test for %s as it is not supported", adapter.GetName())
					}
				})
			*/
		})
	}
}

func createSparseDeviceFile(t *testing.T, fsName string) (string, error) {
	t.Helper()

	file, err := os.CreateTemp("", fmt.Sprintf("%s-device-*.img", fsName))
	if err != nil {
		return "", err
	}

	t.Cleanup(func() {
		_ = os.Remove(file.Name())
	})

	// Some tools (for example mkfs.xfs) require large filesystems.
	const sparseDeviceSize = int64(2 * 1024 * 1024 * 1024) // 2 GiB
	if err := file.Truncate(sparseDeviceSize); err != nil {
		_ = file.Close()
		return "", err
	}

	if err := file.Close(); err != nil {
		return "", err
	}

	return file.Name(), nil
}

func testLabelFor(fsName string) string {
	base := strings.ToUpper(fsName)
	if len(base) > 8 {
		base = base[:8]
	}

	if base == "" {
		return "SRAT"
	}

	return "S" + base
}
