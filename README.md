# Gantt Viewer for Github Project

Gantt Viewer for Github Project

## TODO

### Graphql API


```graphql
query MyQuery {
  organization(login: "bitizenwallet") {
    projectsNext(first: 10) {
      pageInfo {
        hasNextPage
      }
      edges {
        node {
          owner {
            id
          }
          public
          title
          viewerCanUpdate
        }
      }
      nodes {
        title
        updatedAt
        items(first: 10) {
          pageInfo {
            hasNextPage
          }
          nodes {
            title
            fieldValues(first: 10) {
              pageInfo {
                hasNextPage
              }
              nodes {
                value
                updatedAt
                projectField {
                  name
                  dataType
                }
              }
            }
            type
            id
            isArchived
            content {
              ... on Issue {
                id
                assignees(first: 10) {
                  pageInfo {
                    hasNextPage
                  }
                  nodes {
                    name
                    avatarUrl
                  }
                }
                url
              }
              ... on DraftIssue {
                id
                assignees(first: 10) {
                  pageInfo {
                    hasNextPage
                  }
                  nodes {
                    name
                    avatarUrl
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
```

### Gantt render

https://github.com/frappe/gantt
