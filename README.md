# Carlos The Curious

[![CircleCI](https://circleci.com/gh/dklassen/CarlosTheCurious/tree/master.svg?style=svg)](https://circleci.com/gh/dklassen/CarlosTheCurious/tree/master)

Welcome dear world to yet another survey slackbot. This started out as an attempt to learn some Golang and the Slack API. I wanted to build a bot that would be used to pose single simple questions to groups or individuals. The goal to drip health and engagement data ( okay and some fun team trivia) from Slack. Yes, this has be done before but there is nothing like writing something for yourself to learn.
This is still a work in progress but the conversation flow goes like this. Open a private message with Carlos:

```
dana [10:39 PM]  
create response poll 

carlos_the_curiousBOT [10:39 PM]  
Creating response poll. You can cancel the poll any time with `cancel poll 437c67df-febb-4350-b561-92296ea97f6a`

[10:39]  
What was the question you wanted to ask?

dana [10:40 PM]  
Did you remember to write tests first?

carlos_the_curiousBOT [10:40 PM]  
What are the possible responses (comma separated)?

dana [10:40 PM]  
yes,no

carlos_the_curiousBOT [10:40 PM]  
Who should we send this to?

dana [10:41 PM]  
@dana

carlos_the_curiousBOT [10:41 PM]  
Here's a preview of what we are going to send:
Look good to you (yes/no)?
wunderbar
Did you remember to write tests first?
Recipients:
----------------
@dana

Possible Answers:
----------------
yes, no

dana [10:42 PM]  
yes

carlos_the_curiousBOT [10:42 PM]  
We have a question for you. You can answer via `answer poll 437c67df-febb-4350-b561-92296ea97f6a {insert response}`
Question
Did you remember to write tests first?
Possible Answers:
----------------
yes, no

[10:42]  
Poll is live you can check in by asking me to `show poll 437c67df-febb-4350-b561-92296ea97f6a
```

Lots left to do:
- [ ] - test coverage and quality
- [ ] - refactor and code cleanup
- [ ] - better eventing/statemachineness
- [ ] - recipients can be channels or users
- [ ] - display the results of the poll
- [ ] - integration testing
- [ ] - support open ended questions
- [ ] - platform for analysis
- [ ] - schedule polls e.x ask poll every monday morning

## Development

First before you doing anything setup Docker for Mac. Once up and running You can build the project via `make build`. This will build the project inside the container. To use Carlos you will need to have a `postgres` database running somewhere. I would suggest using the postgres docker container which should setup everything at least for development. You can set this up via `make database-build` which will create the container. You can run the container via `make database-up` and bring down with `make database-down`.

Run the container with:
`DATABASE_URL={insert database URL} SLACK_TOKEN={insert your slack token} make run`

But, If you want to trigger the container yourself:
`docker run --net=host --rm -it -e "DATABASE_URL=postgres://postgres:@127.0.0.1/carlos?sslmode=disable" -e "SLACKTOKEN={{insert your slack token here}}" carlos-the-curious`
