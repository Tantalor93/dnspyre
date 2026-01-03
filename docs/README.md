# dnspyre Documentation

## Previewing documentation changes in Pull Requests

When you create a pull request with changes to the documentation (files in the `docs/` directory), a GitHub Actions workflow will automatically:

1. Build the Jekyll site with your changes
2. Deploy a live preview to a unique URL
3. Comment on your PR with a link to view the rendered documentation

This allows reviewers to see exactly how the documentation will look before merging.

The preview will be automatically cleaned up when the PR is closed or merged.

## Building and previewing your site locally

Assuming [Jekyll] and [Bundler] are installed on your computer:

1.  Change your working directory to the root directory of your site.

2.  Run `bundle install`.

3.  Run `bundle exec jekyll serve` to build your site and preview it at `localhost:4000`.

    The built site is stored in the directory `_site`.
