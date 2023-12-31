# scla-unsubscribe

This is a project to unsubscribe all UTSA students from the SCLA mailing list.

The SCLA is one of those membership organizations that email you throughout the semester to join their honor society. Except, it's not a real honor society - it's just a money grab to make you feel special while providing no real value.

This project entails a somewhat advanced Go 'script' to login to UTSA, scrape every single email address, and then unsubscribe them from the SCLA mailing list.

Multiple levels of caching are used to prevent unnecessary requests to the UTSA website.
- Cache cookies, in order to re-use logins across sessions
- Cache A-Z directory pages, as they take a very long time to generate
- Cache individual entries, as there are thousands of them and should never change for our purposes
- Mark emails as unsubscribed, so we don't have to re-unsubscribe them or iterate over them again afterwards

## Pipeline

- Mass Letter Directories
- Individual Directory Entries
- Unsubscribe from SCLA

## Potential Improvements

- Tooling for clearing or viewing the cache database
- Tooling for clearing the unsubscribed database
- Long Directory TTLs
- Railway Deployments (Cron'd)
    - Run once a day, check for expired databases

For all this effort, I have no real idea that SCLA will stop emailing anyone.