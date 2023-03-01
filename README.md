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
Here is an example, assuming you prefer vim as your editor:

```shell
# A few environment variables need to be set; put them in your
# ~/.bashrc, a small wrapper script (e.g. if you want to use a
# password manager for the token) or similar for ease of use.
export EDITOR=vim
export JIRA_USER=your.username@somecorp.com
# Generate a token at https://id.atlassian.com/manage-profile/security/api-tokens
export JIRA_TOKEN=<your-REST-API-token>

# Edit a comment (copy this URL by clicking the chain-symbol on the comment):
jirae 'https://somecorp.atlassian.net/browse/SCO-1234?focusedCommentId=4321'
```

# Tips and Tricks
On most operating systems you can use the `xclip` tool to automatically
read a copied URL from the clipboard. This way you don't have to paste
the URL. Add this alias to your `~/.bashrc` (or similar):
`alias jirae='jirae "$(xclip -o -selection clipboard)"'`. Now you can
simply call `jirae` (without any argument) after you have copied a URL
from Jira.
