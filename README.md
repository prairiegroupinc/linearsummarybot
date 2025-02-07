# Linear Issues Summary Bot

Provides a high-level overview of Linear issues planned for a given month.

Solves these goals:

1. Avoid being overcommitted. (I.e. see how close we are to capacity on a given month.)
2. Balance client work, internals work and strategically important features. (See how much work is allocated towards each.)
3. "What's that strategic work we're doing in March exactly?"


## Understanding the output

First of all, we skip all completed and cancelled issues, and issues without an estimate or estimated at 0 points.

The primary output is a table displaying total points grouped by month, initiative and project.

All issues are split into 3 categories:

* Fixed: issues added to a cycle and having a due date before (cycle end + 14 days). By our conventions, these issues are fixed commitments, and moving them must be negotiated with the business team.
* Planned: all other issues added to a cycle. These are basically "arbitrarily planned to be done in a given week", and can be rescheduled within reason.
* Flex: issues with a due date but not yet added to a cycle. These represent the date we roughly want to do them by, but the date isn't a hard commitment and can be moved within reason.
