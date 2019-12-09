package buildpackage_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/buildpack/imgutil/fakes"
	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpack/pack/internal/api"
	"github.com/buildpack/pack/internal/buildpackage"
	"github.com/buildpack/pack/internal/dist"
	ifakes "github.com/buildpack/pack/internal/fakes"
	h "github.com/buildpack/pack/testhelpers"
	"github.com/buildpack/pack/testmocks"
)

func TestPackageBuilder(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "PackageBuilder", testPackageBuilder, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testPackageBuilder(t *testing.T, when spec.G, it spec.S) {
	var (
		fakePackageImage *fakes.Image
		mockController   *gomock.Controller
		mockImageFactory *testmocks.MockImageFactory
		subject          *buildpackage.PackageBuilder
	)

	it.Before(func() {
		mockController = gomock.NewController(t)
		mockImageFactory = testmocks.NewMockImageFactory(mockController)

		fakePackageImage = fakes.NewImage("some/package", "", nil)
		mockImageFactory.EXPECT().NewImage("some/package", true).Return(fakePackageImage, nil).AnyTimes()

		subject = buildpackage.NewBuilder(mockImageFactory)
	})

	it.After(func() {
		mockController.Finish()
	})

	when("#Save", func() {
		when("validate default", func() {
			when("default not set", func() {
				it("returns error", func() {
					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "a default buildpack must be set")
				})
			})

			when("default is missing from buildpacks", func() {
				it("returns error", func() {
					subject.SetDefaultBuildpack(dist.BuildpackInfo{
						ID:      "bp.1.id",
						Version: "bp.1.version",
					})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "selected default 'bp.1.id@bp.1.version' is not present")
				})
			})
		})

		when("validate stacks", func() {
			it.Before(func() {
				buildpack, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
					API: api.MustParse("0.2"),
					Info: dist.BuildpackInfo{
						ID:      "bp.1.id",
						Version: "bp.1.version",
					},
					Stacks: []dist.Stack{
						{ID: "stack.id.1", Mixins: []string{"Mixin-A"}},
					},
					Order: nil,
				}, 0644)
				h.AssertNil(t, err)

				subject.SetDefaultBuildpack(dist.BuildpackInfo{
					ID:      buildpack.Descriptor().Info.ID,
					Version: buildpack.Descriptor().Info.Version,
				})

				subject.AddBuildpack(buildpack)
			})

			when("no stacks are set", func() {
				it("returns error", func() {
					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "must specify at least one supported stack")
				})
			})

			when("stack is added more than once", func() {
				it("should error", func() {
					subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})
					subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err, "stack 'stack.id.1' was specified more than once")
				})
			})

			when("stack is not listed in bp", func() {
				it("should error", func() {
					subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})
					subject.AddStack(dist.Stack{ID: "stack.id.not-supported-by-bps"})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err,
						"buildpack 'bp.1.id@bp.1.version' does not support stack 'stack.id.not-supported-by-bps'",
					)
				})
			})

			when("stack mixins do not satisfy bp", func() {
				it("should error", func() {
					subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-B"}})

					_, err := subject.Save(fakePackageImage.Name(), false)
					h.AssertError(t, err,
						"buildpack 'bp.1.id@bp.1.version' requires missing mixin(s): Mixin-A",
					)
				})
			})

			when("bp has more supported stacks than package supports", func() {
				it("should be successful", func() {
					buildpack2, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
						API: api.MustParse("0.2"),
						Info: dist.BuildpackInfo{
							ID:      "buildpack.2.id",
							Version: "buildpack.2.version",
						},
						Stacks: []dist.Stack{
							{ID: "stack.id.1"},
							{ID: "stack.id.2"},
						},
						Order: nil,
					}, 0644)
					h.AssertNil(t, err)

					subject.AddBuildpack(buildpack2)
					subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})

					_, err = subject.Save(fakePackageImage.Name(), false)
					h.AssertNil(t, err)
				})
			})
		})

		it("sets metadata", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				API: api.MustParse("0.2"),
				Info: dist.BuildpackInfo{
					ID:      "bp.1.id",
					Version: "bp.1.version",
				},
				Stacks: []dist.Stack{
					{ID: "stack.id.1"},
					{ID: "stack.id.2"},
				},
				Order: nil,
			}, 0644)
			h.AssertNil(t, err)

			subject.AddBuildpack(buildpack1)
			subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})
			subject.SetDefaultBuildpack(dist.BuildpackInfo{
				ID:      "bp.1.id",
				Version: "bp.1.version",
			})

			packageImage, err := subject.Save(fakePackageImage.Name(), false)
			h.AssertNil(t, err)

			labelData, err := packageImage.Label("io.buildpacks.buildpackage.metadata")
			h.AssertNil(t, err)
			var md buildpackage.Metadata
			h.AssertNil(t, json.Unmarshal([]byte(labelData), &md))

			h.AssertEq(t, md.ID, "bp.1.id")
			h.AssertEq(t, md.Version, "bp.1.version")
			h.AssertEq(t, len(md.Stacks), 1)
			h.AssertEq(t, md.Stacks[0].ID, "stack.id.1")
		})

		it("sets buildpack layers label", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				API:    api.MustParse("0.2"),
				Info:   dist.BuildpackInfo{ID: "bp.1.id", Version: "bp.1.version"},
				Stacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				Order:  nil,
			}, 0644)
			h.AssertNil(t, err)
			subject.AddBuildpack(buildpack1)
			subject.SetDefaultBuildpack(dist.BuildpackInfo{ID: "bp.1.id", Version: "bp.1.version"})

			subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})
			_, err = subject.Save(fakePackageImage.Name(), false)
			h.AssertNil(t, err)

			var bpLayers dist.BuildpackLayers
			_, err = dist.GetLabel(fakePackageImage, "io.buildpacks.buildpack.layers", &bpLayers)
			h.AssertNil(t, err)

			bp1Info, ok1 := bpLayers["bp.1.id"]["bp.1.version"]
			h.AssertEq(t, ok1, true)
			h.AssertEq(t, bp1Info.Stacks, []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}})
		})

		it("adds buildpack layers", func() {
			buildpack1, err := ifakes.NewFakeBuildpack(dist.BuildpackDescriptor{
				API:    api.MustParse("0.2"),
				Info:   dist.BuildpackInfo{ID: "bp.1.id", Version: "bp.1.version"},
				Stacks: []dist.Stack{{ID: "stack.id.1"}, {ID: "stack.id.2"}},
				Order:  nil,
			}, 0644)
			h.AssertNil(t, err)
			subject.AddBuildpack(buildpack1)
			subject.SetDefaultBuildpack(dist.BuildpackInfo{ID: "bp.1.id", Version: "bp.1.version"})

			subject.AddStack(dist.Stack{ID: "stack.id.1", Mixins: []string{"Mixin-A"}})
			_, err = subject.Save(fakePackageImage.Name(), false)
			h.AssertNil(t, err)

			buildpackExists := func(name, version string) {
				dirPath := fmt.Sprintf("/cnb/buildpacks/%s/%s", name, version)
				layerTar, err := fakePackageImage.FindLayerWithPath(dirPath)
				h.AssertNil(t, err)

				h.AssertOnTarEntry(t, layerTar, dirPath,
					h.IsDirectory(),
				)

				h.AssertOnTarEntry(t, layerTar, dirPath+"/bin/build",
					h.ContentEquals("build-contents"),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644),
				)

				h.AssertOnTarEntry(t, layerTar, dirPath+"/bin/detect",
					h.ContentEquals("detect-contents"),
					h.HasOwnerAndGroup(0, 0),
					h.HasFileMode(0644),
				)
			}

			buildpackExists("bp.1.id", "bp.1.version")
		})
	})
}
