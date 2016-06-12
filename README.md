# Carlos The Curious

## Status: NOT WORKING

Welcome dear world to yet another survey slackbot. This started out as an attempt to learn some Golang and the Slack API. I wanted to build a bot that would be used to pose single simple questions to groups or individuals. The goal to drip health and engagement data ( okay and some fun team trivia) from Slack. Yes, this has be done before but there is nothing like writing something for yourself to learn.
This is still a work in progress but the conversation flow goes like this. Open a private message with Carlos:

```
create poll wunderbar

Okay, what was the question you wanted to ask?

How many is too many?

What are the possible responses ( comma separated list)

1-5,6-10,10-20

Who would you like to send this too?

@jimmmy,@bobby,@lisa

Heres a preview of the poll:
...
ready to send it? (yes/no)

yes

poll was sent. You can check responses with `show poll wunderbar`
```

After the poll is ready you can send it to those that were listed. You can wait for the responses or finish the poll early and view the results:

```
show results wunderbar
```

Carlos will display the results of your poll and voila you have your answer. There are lots of survey types and going forward we want to be able to ask other closed/open types of questions. Also, we want to build a back end to support some analysis of the responses that you get back.

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
