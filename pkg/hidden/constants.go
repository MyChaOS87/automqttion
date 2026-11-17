package hidden

const HiddenText = "*HIDDEN*"

//nolint:gosec // No hardcoded credential here; regex matches password in connection URLs.
const URLPasswordRegexp = `.*:(?P<password>[^/^@]+)@.*`
