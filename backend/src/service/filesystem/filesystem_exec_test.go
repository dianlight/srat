package filesystem_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dianlight/srat/dto"
	"github.com/dianlight/srat/internal/osutil"
	"github.com/dianlight/srat/service/filesystem"
	"github.com/stretchr/testify/require"
	"github.com/u-root/u-root/pkg/mount/loop"
)

func TestFilesystemUtilsWithoutMockedCommands(t *testing.T) {
	registry := filesystem.NewRegistry()
	adapters := registry.GetAll()

	if len(adapters) == 0 {
		t.Fatal("expected at least one filesystem adapter in registry")
	}

	for _, adapter := range adapters {
		t.Run(adapter.GetName(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			support, supportErr := adapter.IsSupported(ctx)
			if supportErr != nil {
				t.Fatalf("IsSupported failed for %s: %v", adapter.GetName(), supportErr)
			}

			dir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current working directory: %v", err)
			}
			// check if exists a test device image in backend/test and use it instead of creating a new one, to speed up tests and reduce resource usage and avoid issues with some filesystems that don't work well with sparse files (like hfsplus)
			deviceFile := fmt.Sprintf("%s/../../../test/data/%s_test_device.dmg", dir, adapter.GetName())
			if _, err := os.Stat(deviceFile); err == nil {
				t.Logf("using existing test device image for %s: %s", adapter.GetName(), deviceFile)
			} else if support.CanFormat {
				t.Logf("creating new test device image for %s", adapter.GetName())
				testDevicePath, supportErr := createSparseDeviceFile(t, adapter.GetName())
				if supportErr != nil {
					t.Fatalf("failed to create test device file for %s: %v", adapter.GetName(), supportErr)
				}
				deviceFile = testDevicePath
			} else {
				t.Skipf("skipping tests for %s as it is not supported and no test device image (%s) is available", adapter.GetName(), deviceFile)
				return
			}

			device, err := loop.FindDevice()
			if err != nil {
				t.Skip("No loop device available, skipping test")
				return
			}
			require.NoError(t, err, "Error finding loop device")
			err = osutil.CreateBlockDevice(t.Context(), device)
			require.NoError(t, err, "Error creating block device")
			err = loop.SetFile(device, deviceFile)
			require.NoError(t, err, "Error setting loop device file")
			t.Logf("Block device %s binded to %s", device, deviceFile)
			defer func() {
				err := loop.ClearFile(device)
				if err != nil {
					t.Logf("Error clearing loop device file for %s: %v", adapter.GetName(), err)
				}
			}()

			label := testLabelFor(adapter.GetName())

			t.Run(adapter.GetName()+"_format", func(t *testing.T) {
				if support.CanFormat {
					formatStep := []string{"start", "running", "success", "failure"}

					formatErr := adapter.Format(ctx, device, dto.FormatOptions{
						Label: label,
						Force: true,
					}, func(status string, percentage int, notes []string) {
						t.Logf("Format progress for %s: status=%s, percentage=%d, notes=%v", adapter.GetName(), status, percentage, notes)
						if status == "failure" {
							t.Errorf("Format failed for %s: %v", adapter.GetName(), notes)
						}
						if percentage != 999 && (percentage < 0 || percentage > 100) {
							t.Errorf("Invalid percentage value for %s: %d", adapter.GetName(), percentage)
						}
						if status == "success" && percentage == 100 && formatStep[0] == "running" {
							formatStep = formatStep[1:]
						}
						if status != formatStep[0] {
							t.Errorf("Unexpected status for %s: got %s, expected %s", adapter.GetName(), status, formatStep[0])
						}
						if status == "start" {
							formatStep = formatStep[1:] // move to next expected step
						} else if status == "running" && percentage == 999 {
							// allow multiple running updates with percentage 999 (indeterminate progress)
						} else if status == "running" {
							if percentage < 0 || percentage > 100 {
								t.Errorf("Invalid percentage value for %s: %d", adapter.GetName(), percentage)
							}
						} else if status == "success" {
							if percentage != 100 {
								t.Errorf("Success status should have 100%% percentage for %s, got %d", adapter.GetName(), percentage)
							}
						} else if status == "failure" {
							if percentage != 0 {
								t.Errorf("Failure status should have 0%% percentage for %s, got %d", adapter.GetName(), percentage)
							}
						}
					})
					if formatErr != nil {
						t.Fatalf("format attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, formatErr)
					}
				} else {
					t.Skipf("skipping format test for %s as it is not supported", adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_magic", func(t *testing.T) {
				yes, err := adapter.IsDeviceSupported(t.Context(), device)
				require.NoError(t, err, "Error checking device support for %s: %v", adapter.GetName(), err)
				if yes {
					t.Logf("Device %s correctly identified as supported by %s", device, adapter.GetName())
				} else {
					t.Errorf("Device %s not identified as supported by %s", device, adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_state", func(t *testing.T) {
				if support.CanGetState {
					_, stateErr := adapter.GetState(ctx, device)
					if stateErr != nil {
						t.Logf("state attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, stateErr)
					}
				} else {
					t.Skipf("skipping state test for %s as it is not supported", adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_check", func(t *testing.T) {
				if support.CanCheck {
					checkStep := []string{"start", "running", "success", "failure"}
					_, checkErr := adapter.Check(ctx, device, dto.CheckOptions{
						AutoFix: false,
						Force:   true,
						Verbose: false,
					},
						func(status string, percentage int, notes []string) {
							t.Logf("Check progress for %s: status=%s, percentage=%d, notes=%v", adapter.GetName(), status, percentage, notes)
							if status == "failure" {
								t.Errorf("Check failed for %s: %v", adapter.GetName(), notes)
							}
							if percentage != 999 && (percentage < 0 || percentage > 100) {
								t.Errorf("Invalid percentage value for %s: %d", adapter.GetName(), percentage)
							}
							if status == "success" && percentage == 100 && checkStep[0] == "running" {
								checkStep = checkStep[1:]
							}
							if status != checkStep[0] {
								t.Errorf("Unexpected status for %s: got %s, expected %s", adapter.GetName(), status, checkStep[0])
							}
							if status == "start" {
								checkStep = checkStep[1:] // move to next expected step
							} else if status == "running" && percentage == 999 {
								// allow multiple running updates with percentage 999 (indeterminate progress)
							} else if status == "running" {
								if percentage < 0 || percentage > 100 {
									t.Errorf("Invalid percentage value for %s: %d", adapter.GetName(), percentage)
								}
							} else if status == "success" {
								if percentage != 100 {
									t.Errorf("Success status should have 100%% percentage for %s, got %d", adapter.GetName(), percentage)
								}
							} else if status == "failure" {
								if percentage != 0 {
									t.Errorf("Failure status should have 0%% percentage for %s, got %d", adapter.GetName(), percentage)
								}
							}
						},
					)
					if checkErr != nil {
						t.Logf("check attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, checkErr)
					}
				} else {
					t.Skipf("skipping check test for %s as it is not supported", adapter.GetName())
				}
			})

			t.Run(adapter.GetName()+"_label", func(t *testing.T) {
				if support.CanSetLabel {
					setLabelErr := adapter.SetLabel(ctx, device, label)
					if setLabelErr != nil {
						t.Logf("set label attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, setLabelErr)
					}
					getLabel, getLabelErr := adapter.GetLabel(ctx, device)
					if getLabelErr != nil {
						t.Errorf("get label attempt failed for %s on %s: %v", adapter.GetName(), deviceFile, getLabelErr)
					} else if getLabel != label {
						t.Errorf("label mismatch for %s on %s: expected '%s', got '%s'", adapter.GetName(), deviceFile, label, getLabel)
					}
				} else {
					t.Skipf("skipping set label test for %s as it is not supported", adapter.GetName())
				}
			})

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

					_, mountErr := adapter.Mount(ctx, device, mountPoint, adapter.GetLinuxFsModule(), "", 0, nil)
					if mountErr != nil {
						t.Errorf("mount attempt failed for %s on %s: %v", adapter.GetName(), device, mountErr)
						return
					}

					// unmount after test
					unmountErr := adapter.Unmount(ctx, mountPoint, false, false)
					if unmountErr != nil {
						t.Logf("unmount attempt failed for %s on %s: %v", adapter.GetName(), device, unmountErr)
					}
				} else {
					t.Skipf("skipping mount test for %s as it is not supported", adapter.GetName())
				}
			})

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
