package lib

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SkipList", func() {
	It("skips the repos it lists, whatever the case or spacing", func() {
		skip := ParseSkipList(" syntasso/testing-git-writer-private, Syntasso/backstage-PR-testing-repo ,")
		Expect(skip.Skips("syntasso/testing-git-writer-private")).To(BeTrue())
		Expect(skip.Skips("syntasso/backstage-pr-testing-repo")).To(BeTrue())
		Expect(skip.Skips("syntasso/kratix")).To(BeFalse())
	})

	It("skips nothing when the list is empty", func() {
		Expect(ParseSkipList("").Skips("syntasso/kratix")).To(BeFalse())
	})
})
