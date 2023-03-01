`jirae` is a small tool, that allows you to edit Jira comments with your
favorite editor. It has been created, because Atlassian started to force
its users to use their hideous WYSIWYG editor. It is a workaround for
[JRACLOUD-72631](https://jira.atlassian.com/browse/JRACLOUD-72631).

Note, that you must use the "old syntax" of Jira, not
Markdown. E.g. preformatted text is written as `{{text}}`,
headings are written as `h3. Heading`, etc. The full
documentation of this markup language can be found
[here](https://jira.atlassian.com/secure/WikiRendererHelpAction.jspa?section=all).

# Installation
You can find precompiled binaries at the
[releases page](https://github.com/codesoap/jirae/releases). If you
prefer to install from source, execute this:

```
go install github.com/codesoap/jirae@latest
```

# Usage
Here is an example, assuming you work at "somecorp" and prefer vim as
your editor:

```shell
# A few environment variables need to be set; put them in your
# ~/.bashrc, a small wrapper script (e.g. if you want to use a
# password manager for the token) or similar for ease of use.
export EDITOR=vim
export JIRA_URL=https://somecorp.atlassian.net
export JIRA_USER=your.username@somecorp.com
# Generate a token at https://id.atlassian.com/manage-profile/security/api-tokens
export JIRA_TOKEN=<your-REST-API-token>

# Edit the latest comment of the issue with the ID 'SCO-1234':
jirae SCO-1234

# Edit the comment with the ID '4321' of issue 'SCO-1234':
jirae SCO-1234 4321
```
