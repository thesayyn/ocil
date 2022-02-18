/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"errors"
	"fmt"
	"log"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
func NewPullCmd() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "A brief description of your application",
		RunE:  Run,
		Args:  cobra.ExactArgs(2),
	}
	pullCmd.Flags().String("select", "", "")
	pullCmd.Flags().Bool("flatten", false, "Convert ")
	return pullCmd
}

func Run(cmd *cobra.Command, args []string) error {
	src := args[0]
	dest := args[1]
	slct, err := cmd.Flags().GetString("select")

	o := crane.GetOptions()

	ref, err := name.ParseReference(src, o.Name...)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %w", src, err)
	}

	rmt, err := remote.Get(ref, o.Remote...)
	if err != nil {
		return err
	}

	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("mediaType", decls.String),
			decls.NewVar("size", decls.Int),
			decls.NewVar("digest", decls.Int),
			decls.NewVar("data", decls.Bytes),
			decls.NewVar("urls", decls.NewListType(decls.String)),
			decls.NewVar("annotations", decls.NewMapType(decls.String, decls.String)),
			decls.NewVar("platform.os", decls.String),
			decls.NewVar("platform.architecture", decls.String),
			decls.NewVar("platform.variant", decls.String),
			decls.NewVar("platform.os.version", decls.String),
			decls.NewVar("platform.os.features", decls.NewListType(decls.String)),
			decls.NewVar("platform.features", decls.NewListType(decls.String)),
		),
	)

	ast, issues := env.Compile(slct)

	if len(issues.Errors()) > 0 {
		return issues.Err()
	}

	p, err := env.Program(ast)

	if err != nil {
		return err
	}

	if rmt.MediaType.IsIndex() {
		idx, err := rmt.ImageIndex()
		if err != nil {
			return err
		}
		imn, err := idx.IndexManifest()
		if err != nil {
			return err
		}

		rm := []v1.Hash{}

		for i, manifest := range imn.Manifests {

			r, _, err := p.Eval(map[string]interface{}{
				"mediaType":             manifest.MediaType,
				"size":                  manifest.Size,
				"digest":                manifest.Digest.String(),
				"data":                  manifest.Data,
				"urls":                  manifest.URLs,
				"annotations":           manifest.Annotations,
				"platform.os":           manifest.Platform.OS,
				"platform.architecture": manifest.Platform.Architecture,
				"platform.variant":      manifest.Platform.Variant,
				"platform.os.version":   manifest.Platform.OSVersion,
				"platform.os.features":  manifest.Platform.OSFeatures,
				"platform.features":     manifest.Platform.Features,
			})

			if err != nil {
				return err
			}

			if r != types.True {
				rm = append(rm, manifest.Digest)
				log.Printf("skipping image %d as it does not match --select", i)
				continue
			}
			log.Printf("picking image [%d] %s/%s", i, manifest.Platform.OS, manifest.Platform.Architecture)
		}

		if err := crane.MultiSaveOCI(map[string]v1.Image{}, dest); err != nil {
			return fmt.Errorf("saving oci image layout %s: %w", dest, err)
		}

		p, err := layout.FromPath(dest)
		if err != nil {
			return err
		}

		if err := p.AppendIndex(mutate.RemoveManifests(idx, match.Digests(rm...)), layout.WithAnnotations(map[string]string{
			"org.opencontainers.image.ref.name": ref.String(),
		})); err != nil {
			return err
		}

	} else {
		return errors.New("index only")
	}

	return nil
}
