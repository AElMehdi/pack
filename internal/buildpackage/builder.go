package buildpackage

import (
	"io/ioutil"
	"os"

	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/stack"
	"github.com/buildpacks/pack/internal/style"
)

type ImageFactory interface {
	NewImage(repoName string, local bool) (imgutil.Image, error)
}

type PackageBuilder struct {
	buildpack    dist.Buildpack
	dependencies []dist.Buildpack
	imageFactory ImageFactory
}

func NewBuilder(imageFactory ImageFactory) *PackageBuilder {
	return &PackageBuilder{
		imageFactory: imageFactory,
	}
}

func (p *PackageBuilder) SetBuildpack(buildpack dist.Buildpack) {
	p.buildpack = buildpack
}

func (p *PackageBuilder) AddDependency(buildpack dist.Buildpack) {
	p.dependencies = append(p.dependencies, buildpack)
}

func (p *PackageBuilder) Save(repoName string, publish bool) (imgutil.Image, error) {
	if p.buildpack == nil {
		return nil, errors.New("buildpack must be set")
	}

	// TODO: Do we need to check main buildpack separately?
	// Can we try starting with `stacks := []string{}`, then including
	// `p.buildpack` in the for loop below (line 55)?
	stacks := p.buildpack.Descriptor().Stacks
	if len(stacks) == 0 && len(p.buildpack.Descriptor().Order) == 0 {
		return nil, errors.Errorf(
			"buildpack %s must support at least one stack or have an order",
			style.Symbol(p.buildpack.Descriptor().Info.FullName()),
		)
	}

	for _, bp := range p.dependencies { // TODO: append/prepend `p.buildpack` here instead?
		bpd := bp.Descriptor()
		stacks = stack.MergeCompatible(stacks, bpd.Stacks)
		if len(stacks) == 0 {  // TODO: check order here, only error if len(order) == 0 and len(stacks) == 0
			return nil, errors.Errorf(
				"buildpack %s does not support any stacks from %s",
				style.Symbol(p.buildpack.Descriptor().Info.FullName()),
				style.Symbol(bpd.Info.FullName()),
			)
		}
	}

	image, err := p.imageFactory.NewImage(repoName, !publish)
	if err != nil {
		return nil, errors.Wrapf(err, "creating image")
	}

	if err := dist.SetLabel(image, MetadataLabel, &Metadata{
		BuildpackInfo: p.buildpack.Descriptor().Info,
		Stacks:        stacks,
	}); err != nil {
		return nil, err
	}

	tmpDir, err := ioutil.TempDir("", "create-package")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	bpLayers := dist.BuildpackLayers{}
	for _, bp := range append(p.dependencies, p.buildpack) {
		bpLayerTar, err := dist.BuildpackToLayerTar(tmpDir, bp)
		if err != nil {
			return nil, err
		}

		if err := image.AddLayer(bpLayerTar); err != nil {
			return nil, errors.Wrapf(err, "adding layer tar for buildpack %s", style.Symbol(bp.Descriptor().Info.FullName()))
		}

		diffID, err := dist.LayerDiffID(bpLayerTar)
		if err != nil {
			return nil, errors.Wrapf(err,
				"getting content hashes for buildpack %s",
				style.Symbol(bp.Descriptor().Info.FullName()),
			)
		}

		dist.AddBuildpackToLayersMD(bpLayers, bp.Descriptor(), diffID.String())
	}

	if err := dist.SetLabel(image, dist.BuildpackLayersLabel, bpLayers); err != nil {
		return nil, err
	}

	if err := image.Save(); err != nil {
		return nil, err
	}

	return image, nil
}
