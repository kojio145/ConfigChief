# Security Policy

## Reporting a Vulnerability

Please report vulnerabilities privately via
**[GitHub Security Advisories](../../security/advisories/new)**
("Report a vulnerability" on this repository's Security tab).
Do **not** open a public Issue for security problems.

This is a spare-time project, so acknowledgement and fixes happen on an irregular
schedule, but security reports are treated with the highest priority among all issues.

## Supported Versions

Only the latest release is supported. Older releases do not receive fixes.

## Design Notes

ConfigChief is designed for use inside closed networks:

- No telemetry, no auto-update, no external communication of any kind, no listening port
  (a test in the repository checks that the program's own code calls no network function)
- It never connects to a device: it reads and writes text files and Excel files in a case folder
- Passwords and keys of the configurations are kept as plain text in the parameter book
  (the Excel file of the case), as they are in the configurations themselves. Keep the case
  folder where you keep the configurations. The screens, the check report and the list of
  changes mask these values; the generated configurations and the delivery sheets hold them
- The report of unknown lines that is meant to be carried out of a site
  (`export --what unhandled`) masks values and names, but look at it before you hand it over:
  the program cannot know every word that names a site

## 日本語

脆弱性は、このリポジトリの Security タブの「Report a vulnerability」から、非公開で知らせてください（公開の Issue には書かないでください）。対応するのは最新のリリースだけです。

ConfigChief は外部と通信せず、機器にもつなぎません。コンフィグの中のパスワードや鍵は、パラメータブック（Excel）に平文で入ります。案件フォルダは、コンフィグと同じ扱いで保管してください。
