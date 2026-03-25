# Goal

Over the last few days we have iterated VERY fast on this stock/future/crypto/prediction market trading library and two other repos nearby (brandon-bot-server and brandon-bot-frontend). Over that time, I think we have overfit our solutions to problems at hand, leading to bugs in other areas, and in general have built a good system but one that still feels flimsy to me. I want to walk you through first what each repo is meant to be and then I want you to do a full audit of this repo. Looking at all the logic and primary levers that impact the end user and see if you can find gaps. Without implementing, document what you see, what can be improved, what can be overhauled, what is less important but maybe worth improving or implementing. This product is still in development so breaking changes are ok if necessary.

## brandon-bot (this repo)

This is an opensource trading library with multi-provider support and simulation backtesting. Everything from pulling market data from various providers (topstepx/kalshi/coinbase/tradovate/alpaca/massive), to executing trades, to listening to active market data, and more all lives in this repo. The primary 2 purposes of the library are the following:

1. Run strategies
2. Backtest strategies/Simulate strategies

The way this is accomplished is by parsing a javascript "strategy" input by the end user and using goja to turn it into a running trading loop. The backtest engine and the engine for actually running the trading should function relatively the same logic. Strategies are abstracted so they can use whatever provider/broker they need to and be configured however which way they need to. The strategy controls most of what it needs to function and we expose a lot of global variables to make it easier for the strategy to accomplish what it needs to. The goal is to give the end user who writes the strategy (who we have to assume has little to no knowledge of this repo and it's logic other than like the global variables and required functions and such for the js script to run) as much freedom to design algorithmic trading strategies that will earn them profits. We have to balance this with being relatively provider agnostic (some providers support some functionality others don't so sometimes we use optional interfaces).

## brandon-bot-backend

A backend webserver that primarily serves to consume this brandon-bot library by assisting users in making functional script strategies. The backend uses AI (right now just claude-sonnet-4-6) in an iterative development cycle (plan->consult user->code->backtest->(code/adjust->backtest...)->finalize with configuration) to design the strategies given a system prompt that attempts to explain how to design these js strategies.

Manages the lifecycle of running strategies including what happens when the server restarts (recovery), how do we persist order history, how do we calculate success to show to the end user, etc.

## brandon-bot-frontend

Self-explanatory website to access the backend.

# Conclusion

I need you to be an expert systems designer and software engineer. We need to understand what the goals are and design clean, simple systems that accomplish without bloat and the minimal possibility for errors or unexpected behaviour to occur while being performant and scalable. Review the repo and present your findings.
