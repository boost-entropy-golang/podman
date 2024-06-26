package util

import (
	"io"
	"os"
	"path/filepath"

	"github.com/containers/buildah/define"
	"github.com/containers/common/libimage"
	"github.com/containers/image/v5/types"
	encconfig "github.com/containers/ocicrypt/config"
	enchelpers "github.com/containers/ocicrypt/helpers"
	"github.com/containers/storage"
	"github.com/containers/storage/pkg/archive"
	"github.com/containers/storage/pkg/chrootarchive"
	"github.com/containers/storage/pkg/unshare"
	"github.com/pkg/errors"
)

// LookupImage returns *Image to corresponding imagename or id
func LookupImage(ctx *types.SystemContext, store storage.Store, image string) (*libimage.Image, error) {
	systemContext := ctx
	if systemContext == nil {
		systemContext = &types.SystemContext{}
	}
	runtime, err := libimage.RuntimeFromStore(store, &libimage.RuntimeOptions{SystemContext: systemContext})
	if err != nil {
		return nil, err
	}
	localImage, _, err := runtime.LookupImage(image, nil)
	if err != nil {
		return nil, err
	}
	return localImage, nil
}

// GetTempDir returns base for a temporary directory on host.
func GetTempDir() string {
	if tmpdir, ok := os.LookupEnv("TMPDIR"); ok {
		return tmpdir
	}
	return "/var/tmp"
}

// ExportFromReader reads bytes from given reader and exports to external tar, directory or stdout.
func ExportFromReader(input io.Reader, opts define.BuildOutputOption) error {
	var err error
	if !filepath.IsAbs(opts.Path) {
		opts.Path, err = filepath.Abs(opts.Path)
		if err != nil {
			return err
		}
	}
	if opts.IsDir {
		// In order to keep this feature as close as possible to
		// buildkit it was decided to preserve ownership when
		// invoked as root since caller already has access to artifacts
		// therefore we can preserve ownership as is, however for rootless users
		// ownership has to be changed so exported artifacts can still
		// be accessible by unpriviledged users.
		// See: https://github.com/containers/buildah/pull/3823#discussion_r829376633
		noLChown := false
		if unshare.IsRootless() {
			noLChown = true
		}

		err = os.MkdirAll(opts.Path, 0700)
		if err != nil {
			return errors.Wrapf(err, "failed while creating the destination path %q", opts.Path)
		}

		err = chrootarchive.Untar(input, opts.Path, &archive.TarOptions{NoLchown: noLChown})
		if err != nil {
			return errors.Wrapf(err, "failed while performing untar at %q", opts.Path)
		}
	} else {
		outFile := os.Stdout
		if !opts.IsStdout {
			outFile, err = os.Create(opts.Path)
			if err != nil {
				return errors.Wrapf(err, "failed while creating destination tar at %q", opts.Path)
			}
			defer outFile.Close()
		}
		_, err = io.Copy(outFile, input)
		if err != nil {
			return errors.Wrapf(err, "failed while performing copy to %q", opts.Path)
		}
	}
	return nil
}

// DecryptConfig translates decryptionKeys into a DescriptionConfig structure
func DecryptConfig(decryptionKeys []string) (*encconfig.DecryptConfig, error) {
	decryptConfig := &encconfig.DecryptConfig{}
	if len(decryptionKeys) > 0 {
		// decryption
		dcc, err := enchelpers.CreateCryptoConfig([]string{}, decryptionKeys)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid decryption keys")
		}
		cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{dcc})
		decryptConfig = cc.DecryptConfig
	}

	return decryptConfig, nil
}

// EncryptConfig translates encryptionKeys into a EncriptionsConfig structure
func EncryptConfig(encryptionKeys []string, encryptLayers []int) (*encconfig.EncryptConfig, *[]int, error) {
	var encLayers *[]int
	var encConfig *encconfig.EncryptConfig

	if len(encryptionKeys) > 0 {
		// encryption
		encLayers = &encryptLayers
		ecc, err := enchelpers.CreateCryptoConfig(encryptionKeys, []string{})
		if err != nil {
			return nil, nil, errors.Wrapf(err, "invalid encryption keys")
		}
		cc := encconfig.CombineCryptoConfigs([]encconfig.CryptoConfig{ecc})
		encConfig = cc.EncryptConfig
	}
	return encConfig, encLayers, nil
}

// GetFormat translates format string into either docker or OCI format constant
func GetFormat(format string) (string, error) {
	switch format {
	case define.OCI:
		return define.OCIv1ImageManifest, nil
	case define.DOCKER:
		return define.Dockerv2ImageManifest, nil
	default:
		return "", errors.Errorf("unrecognized image type %q", format)
	}
}
