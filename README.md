# life_notes

## Usage
This application is essentially a notes app with logic.
It's usefulness comes from being personalized and better organized then a regular notepad.
such as a car note where it keeps track of dates, costs, etc and calculates them for you.
All you need to do is log the services and whatnot and it gets posted into the ledger (DB).
It relies on consistant updates for acurate data and calculations, but the goal is for a beautiful
ledger of all the logs and history of the specific note.

Single binary packaged with a web UI. Creates a DB file if there isint one.

## How the program works
- main.go
This is the entry to the program where it first opens/initializes the DB, parses the pages and registers
the endpoints for the pages. Then finally serves on localhost. It contains some fail and close counter measures.

- notes/
This has all of the note logic and handling, for example car.go in here gets called to register its own paths
and endpoints. And has the functions for GET and POST for each table in car. When these functions are called
they call a redirect to refresh the data if everything is success.

- templates/
This folder contains the html layout pages and the note pages. The note pages are registered in main and are
attatched with the HTTP methods in the 'notes/_.go' files so when you say for example reach /car in the browser
we have already attached the GET with the car data information from the database so it sends the template the data
essentially and fills the template's data holes.

## Bugs/ToDo
- Pretty up and design baseplate & home pages
- Implement the rest of car
    - Design UI
    - Implement the rest of the functionality interacting with all car tables in DB
        - Such as Servicing the car, Gas filling, Parts management, more?
- Implement other notes
    - Finance (spending, debts & payroll)
    - Possible course notes? (i write on paper usually so maybe write & store images?)

## Features/Extra
- In main page or something, show a calendar of current month and upcoming dates.
