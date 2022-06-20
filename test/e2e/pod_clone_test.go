package integration

import (
	"os"

	. "github.com/containers/podman/v4/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Podman pod clone", func() {
	var (
		tempdir    string
		err        error
		podmanTest *PodmanTestIntegration
	)

	BeforeEach(func() {
		SkipIfRemote("podman pod clone is not supported in remote")
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		podmanTest = PodmanTestCreate(tempdir)
		podmanTest.Setup()
	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)

	})

	It("podman pod clone basic test", func() {
		create := podmanTest.Podman([]string{"pod", "create", "--name", "1"})
		create.WaitWithDefaultTimeout()
		Expect(create).To(Exit(0))

		run := podmanTest.Podman([]string{"run", "--pod", "1", "-dt", ALPINE})
		run.WaitWithDefaultTimeout()
		Expect(run).To(Exit(0))

		clone := podmanTest.Podman([]string{"pod", "clone", create.OutputToString()})
		clone.WaitWithDefaultTimeout()
		Expect(clone).To(Exit(0))

		podInspect := podmanTest.Podman([]string{"pod", "inspect", clone.OutputToString()})
		podInspect.WaitWithDefaultTimeout()
		Expect(podInspect).To(Exit(0))
		data := podInspect.InspectPodToJSON()
		Expect(data.Name).To(ContainSubstring("-clone"))

		podStart := podmanTest.Podman([]string{"pod", "start", clone.OutputToString()})
		podStart.WaitWithDefaultTimeout()
		Expect(podStart).To(Exit(0))

		podInspect = podmanTest.Podman([]string{"pod", "inspect", clone.OutputToString()})
		podInspect.WaitWithDefaultTimeout()
		Expect(podInspect).To(Exit(0))
		data = podInspect.InspectPodToJSON()
		Expect(data.Containers[1].State).To(ContainSubstring("running")) // make sure non infra ctr is running
	})

	It("podman pod clone renaming test", func() {
		create := podmanTest.Podman([]string{"pod", "create", "--name", "1"})
		create.WaitWithDefaultTimeout()
		Expect(create).To(Exit(0))

		clone := podmanTest.Podman([]string{"pod", "clone", "--name", "2", create.OutputToString()})
		clone.WaitWithDefaultTimeout()
		Expect(clone).To(Exit(0))

		podInspect := podmanTest.Podman([]string{"pod", "inspect", clone.OutputToString()})
		podInspect.WaitWithDefaultTimeout()
		Expect(podInspect).To(Exit(0))
		data := podInspect.InspectPodToJSON()
		Expect(data.Name).To(ContainSubstring("2"))

		podStart := podmanTest.Podman([]string{"pod", "start", clone.OutputToString()})
		podStart.WaitWithDefaultTimeout()
		Expect(podStart).To(Exit(0))
	})

	It("podman pod clone start test", func() {
		create := podmanTest.Podman([]string{"pod", "create", "--name", "1"})
		create.WaitWithDefaultTimeout()
		Expect(create).To(Exit(0))

		clone := podmanTest.Podman([]string{"pod", "clone", "--start", create.OutputToString()})
		clone.WaitWithDefaultTimeout()
		Expect(clone).To(Exit(0))

		podInspect := podmanTest.Podman([]string{"pod", "inspect", clone.OutputToString()})
		podInspect.WaitWithDefaultTimeout()
		Expect(podInspect).To(Exit(0))
		data := podInspect.InspectPodToJSON()
		Expect(data.State).To(ContainSubstring("Running"))

	})

	It("podman pod clone destroy test", func() {
		create := podmanTest.Podman([]string{"pod", "create", "--name", "1"})
		create.WaitWithDefaultTimeout()
		Expect(create).To(Exit(0))

		clone := podmanTest.Podman([]string{"pod", "clone", "--destroy", create.OutputToString()})
		clone.WaitWithDefaultTimeout()
		Expect(clone).To(Exit(0))

		podInspect := podmanTest.Podman([]string{"pod", "inspect", create.OutputToString()})
		podInspect.WaitWithDefaultTimeout()
		Expect(podInspect).ToNot(Exit(0))
	})

	It("podman pod clone infra option test", func() {
		// proof of concept that all currently tested infra options work since

		volName := "testVol"
		volCreate := podmanTest.Podman([]string{"volume", "create", volName})
		volCreate.WaitWithDefaultTimeout()
		Expect(volCreate).Should(Exit(0))

		podName := "testPod"
		podCreate := podmanTest.Podman([]string{"pod", "create", "--name", podName})
		podCreate.WaitWithDefaultTimeout()
		Expect(podCreate).Should(Exit(0))

		podClone := podmanTest.Podman([]string{"pod", "clone", "--volume", volName + ":/tmp1", podName})
		podClone.WaitWithDefaultTimeout()
		Expect(podClone).Should(Exit(0))

		podInspect := podmanTest.Podman([]string{"pod", "inspect", "testPod-clone"})
		podInspect.WaitWithDefaultTimeout()
		Expect(podInspect).Should(Exit(0))
		data := podInspect.InspectPodToJSON()
		Expect(data.Mounts[0]).To(HaveField("Name", volName))
	})

})
