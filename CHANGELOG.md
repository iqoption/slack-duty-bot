CHANGELOG
=========

1.3.0
-----
 * Add correct validation fo duties list in config
 * Add possibility to use only `slack.group` in configuration
 * Fix panic `Out of range` on empty duties list in config
 * Fix `.gitignore` file (to ignore compiled binary)
 * Update `Makefile` (remove useless checks)

1.2.0
-----
 * Using go mod instead of dep [#8 - Use go mod instead of dep](//github.com/iqoption/slack-duty-bot/issues/8)
 * Update `Makefile`
 * Fix bug with setting `Debug` logger level
 * Add utils: bash script for preparing release artifacts
 * Add helm support [#9 - Use helm for install in kubernetes](//github.com/iqoption/slack-duty-bot/issues/9)

1.1.0
-----
 * Fixed issue [#4 - Kubernetes changed config map does not trigger fs event bug](//github.com/iqoption/slack-duty-bot/issues/4)
 * Fixed issue [#5 - Bot gets triggered by channel topic modification bug](//github.com/iqoption/slack-duty-bot/issues/5)
 * Add travis integration for automatic builds
 
1.0.0
-----
 * Using `Slack RTM`
 * Configuration with config file, environment variables and flags
 * Live reload config file
 * Docker image
 * Simple Kubernetes deploy

Init
----
 * Simple `slack-duty-bot` working with http outgoing slack webhook
