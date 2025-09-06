Aggrogator, a feed tracking program for multiple users.

This uses some external tools in order to work:
- Go
- Postgres

The Aggrogator can be installed with the command:
```
go install github.com/itstwoam/aggrogator
```

A database needs to be set up and it's information passed into a config file (.gatorconfig) which should be located in the home directory.

It's contents should look like the following:
```
{
  "db_url": "postgres://<username>:<password>@localhost:<port>/<database name>?sslmode=disable"
}
```

The aggrogator keeps tracks of feeds supplied to it by all users.  Users can add feeds to be followed and follow feeds added by other users.

To get started register at least one user with:
```
aggrogator register <username>
```
The supplied name is now registered and logged in.

Since multiple users may be registered the current user can login with the command:
```
aggrogator login <username>
```
Now the username is logged in.

To add a new feed use the command:
```
aggrogator addfeed "<name>" "<url>"
```
Both name and url must be present and surrounded by double quotes(").  This will add the feed to the list of available feeds to follow and automatically follow that feed for the logged in user.

You can browse feeds with:
```
aggrogator feeds
```
This will let you see the available feeds to be followed

Currently followed feeds may be seen with:
```
aggrogator following
```
This will list the currently followed feeds.

There are many more commands to use and they can be seen with:
```
aggrogator help
```
