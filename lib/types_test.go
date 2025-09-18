package lib

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TypeMapping", func() {
	var typeMapping *TypeMapping

	BeforeEach(func() {
		typeMapping = NewTypeMapping()
	})

	Describe("GetTypeFromTitle", func() {
		Context("when the title has a valid conventional commit prefix", func() {
			It("should return the correct type for basic prefixes", func() {
				testCases := map[string]string{
					"feat: add new feature":       "Feature",
					"bug: fix issue":              "Bug",
					"docs: update documentation":  "Docs",
					"blog: write blog post":       "Blog",
					"interrupt: urgent fix":       "Interrupt",
					"spike: investigate solution": "Spike",
					"chore: maintenance task":     "Chore",
				}

				for title, expectedType := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeTrue(), "Expected to find type for title: %s", title)
					Expect(typeName).To(Equal(expectedType), "Expected type %s for title: %s", expectedType, title)
				}
			})

			It("should return the correct type for prefixes with brackets", func() {
				testCases := map[string]string{
					"feat(ske-operator): add new feature":    "Feature",
					"feat(api): add new feature":             "Feature",
					"bug(frontend): fix issue":               "Bug",
					"bug(backend): fix issue":                "Bug",
					"docs(api): update documentation":        "Docs",
					"docs(user-guide): update documentation": "Docs",
					"blog(announcement): write blog post":    "Blog",
					"interrupt(urgent): urgent fix":          "Interrupt",
					"spike(research): investigate solution":  "Spike",
					"chore(ci): maintenance task":            "Chore",
					"chore(deps): maintenance task":          "Chore",
				}

				for title, expectedType := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeTrue(), "Expected to find type for title: %s", title)
					Expect(typeName).To(Equal(expectedType), "Expected type %s for title: %s", expectedType, title)
				}
			})

			It("should handle prefixes with multiple brackets", func() {
				testCases := map[string]string{
					"feat(api)(v2): add new feature":      "Feature",
					"bug(frontend)(ui): fix issue":        "Bug",
					"docs(api)(v1): update documentation": "Docs",
				}

				for title, expectedType := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeTrue(), "Expected to find type for title: %s", title)
					Expect(typeName).To(Equal(expectedType), "Expected type %s for title: %s", expectedType, title)
				}
			})

			It("should handle case insensitive prefixes", func() {
				testCases := map[string]string{
					"FEAT: add new feature":      "Feature",
					"Feat: add new feature":      "Feature",
					"BUG: fix issue":             "Bug",
					"Bug: fix issue":             "Bug",
					"DOCS: update documentation": "Docs",
					"Docs: update documentation": "Docs",
					"FEAT(API): add new feature": "Feature",
					"Feat(Api): add new feature": "Feature",
				}

				for title, expectedType := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeTrue(), "Expected to find type for title: %s", title)
					Expect(typeName).To(Equal(expectedType), "Expected type %s for title: %s", expectedType, title)
				}
			})

			It("should handle whitespace around prefixes", func() {
				testCases := map[string]string{
					" feat: add new feature":       "Feature",
					"feat : add new feature":       "Feature",
					" feat : add new feature":      "Feature",
					" feat(api): add new feature":  "Feature",
					"feat(api) : add new feature":  "Feature",
					" feat(api) : add new feature": "Feature",
				}

				for title, expectedType := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeTrue(), "Expected to find type for title: %s", title)
					Expect(typeName).To(Equal(expectedType), "Expected type %s for title: %s", expectedType, title)
				}
			})
		})

		Context("when the title has an invalid or missing prefix", func() {
			It("should return false for titles without colons", func() {
				testCases := []string{
					"add new feature",
					"fix issue",
					"update documentation",
					"feat add new feature",
					"bug fix issue",
				}

				for _, title := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeFalse(), "Expected not to find type for title: %s", title)
					Expect(typeName).To(BeEmpty(), "Expected empty type name for title: %s", title)
				}
			})

			It("should return false for unknown prefixes", func() {
				testCases := []string{
					"invalid: no match",
					"unknown: no match",
					"invalid(scope): no match",
					"unknown(scope): no match",
					"random: no match",
					"random(scope): no match",
				}

				for _, title := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeFalse(), "Expected not to find type for title: %s", title)
					Expect(typeName).To(BeEmpty(), "Expected empty type name for title: %s", title)
				}
			})

			It("should return false for empty or whitespace-only titles", func() {
				testCases := []string{
					"",
					" ",
					"  ",
					"\t",
					"\n",
				}

				for _, title := range testCases {
					typeName, found := typeMapping.GetTypeFromTitle(title)
					Expect(found).To(BeFalse(), "Expected not to find type for title: '%s'", title)
					Expect(typeName).To(BeEmpty(), "Expected empty type name for title: '%s'", title)
				}
			})
		})
	})

	Describe("SetTypeID and GetTypeID", func() {
		It("should set and get type IDs correctly", func() {
			typeMapping.SetTypeID("Feature", "feature-id-123")
			typeMapping.SetTypeID("Bug", "bug-id-456")
			typeMapping.SetTypeID("Docs", "docs-id-789")

			id, found := typeMapping.GetTypeID("Feature")
			Expect(found).To(BeTrue())
			Expect(id).To(Equal("feature-id-123"))

			id, found = typeMapping.GetTypeID("Bug")
			Expect(found).To(BeTrue())
			Expect(id).To(Equal("bug-id-456"))

			id, found = typeMapping.GetTypeID("Docs")
			Expect(found).To(BeTrue())
			Expect(id).To(Equal("docs-id-789"))

			id, found = typeMapping.GetTypeID("Unknown")
			Expect(found).To(BeFalse())
			Expect(id).To(BeEmpty())
		})

		It("should allow updating existing type IDs", func() {
			typeMapping.SetTypeID("Feature", "feature-id-123")

			id, found := typeMapping.GetTypeID("Feature")
			Expect(found).To(BeTrue())
			Expect(id).To(Equal("feature-id-123"))

			typeMapping.SetTypeID("Feature", "feature-id-456")

			id, found = typeMapping.GetTypeID("Feature")
			Expect(found).To(BeTrue())
			Expect(id).To(Equal("feature-id-456"))
		})
	})

	Describe("NewTypeMapping", func() {
		It("should initialize with correct default prefix mappings", func() {
			expectedMappings := map[string]string{
				"feat":      "Feature",
				"bug":       "Bug",
				"docs":      "Docs",
				"blog":      "Blog",
				"interrupt": "Interrupt",
				"spike":     "Spike",
				"chore":     "Chore",
			}

			for prefix, expectedType := range expectedMappings {
				typeName, found := typeMapping.PrefixToType[prefix]
				Expect(found).To(BeTrue(), "Expected prefix %s to be mapped", prefix)
				Expect(typeName).To(Equal(expectedType), "Expected prefix %s to map to %s", prefix, expectedType)
			}
		})

		It("should initialize with empty TypeToID map", func() {
			Expect(typeMapping.TypeToID).To(BeEmpty())
		})
	})
})
