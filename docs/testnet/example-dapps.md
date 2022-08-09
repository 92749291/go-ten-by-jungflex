# Example Dapps on Testnet
Obscuro Testnet includes an example dapp to help you better understand how dapps capitalise on Obscuro's unique privacy features.

## Number Guessing Game
[A number guessing game](http://testnet-obscuroscan.uksouth.azurecontainer.io/game.html) is a simple way of showcasing the principles of privacy in Obscuro. The goal of the game is to guess a secret number. Each time an attempt is made an entrance fee of 1 token is paid. If a player correctly guesses the number the contract will pay out all of the accumulated entrance fees to them and reset itself with a new random number.

Consider for a moment how a secret number could be created without divulging it. This is impossible in a transparent, decentralised ecosystem like Ethereum because when the secret number is generated by the smart contract it would be trivial to look up the internal state of the contract and see the secret number. OK, so let's imagine the internal state of the contract and therefore the secret number _is_ hidden, what happens when players start making their guesses? Using a block explorer to see the attempted guesses in plain view within the details of the transaction immediately provides new players with an unfair advantage and the game is ruined.

Building the guessing game in Obscuro addresses both scenarios described above. The guessing game smart contract generates a random secret number when it is deployed. This number is never revealed to a player or node operator, or indeed anybody because it is generated within a Trusted Execution Environment. And when players make their guesses the transactions carrying the guesses are encrypted and therefore obscured in a block explorer.

### How to Play
1. Start up the wallet extension. Follow instructions [here](https://docs.obscu.ro/wallet-extension/wallet-extension.html).
1. Sign in to MetaMask.
1. If this is the first time connecting to Obscuro Testnet then in MetaMask add a new custom network called _Obscuro Testnet_. Add `http://127.0.0.1:3000/` as the "New RPC URL", and use `777` as the "Chain ID" (fill in the other parameters as you see fit).
1. Visit `http://localhost:3000/viewingkeys/` to generate a new viewing key. Connect and sign the viewing key when prompted by MetaMask.
1. For the moment, request some guessing game tokens to play the game by posting a message with your wallet address to [this Discord thread](https://discord.com/channels/916052669955727371/1004391962733989960). 50 game tokens will be sent to your account. Note this is a temporary solution for now.
1. If you want to see this balance in your wallet, you have to import a new Token with the address: ``0xf3a8bd422097bFdd9B3519Eaeb533393a1c561aC``
1. Browse to [the number guessing game](http://testnet-obscuroscan.uksouth.azurecontainer.io/game.html). Check you see `Network ID: 777` at the top of the game window to confirm you are connected to Obscuro Testnet.
1. MetaMask will open and ask to connect your account. Click `Next` then click `Connect`.
1. Approve the payment of 1 token to play the game (this will be added to the prize pool) by clicking the `Approve game fee` button.
1. MetaMask will ask for your account to sign a transaction specifying the Guess contract address as the delegate. Click `Confirm`. Once approved you will see a confirmation popup. Click `OK`.
1. Now you can play the game by typing your guess into the secret number field. Click the `Submit guess` button.
1. MetaMask will ask for your account to sign a contract interaction (your guess). Click `Confirm`. Note how the data presented by MetaMask to you is not yet encypted. That happens when MetaMask signs the transaction and sends it to the wallet extension "network", allowing the wallet extension to encrypt it before it leaves your computer.
1. Open MetaMask to check the progress of your guess transaction in the `Activity` tab of MetaMask. Wait a few moments for it to change from pending to complete.
1. Refresh the guessing game browser window / tab to see if the prize pool has either:
    * Reset to 0. Congratulations! You won the game! Or,
    * Incremented by 1. Unlucky! You did not guess the secret number. Please try again.

Once the guess transaction is completed you can check the guess transaction on ObscuroScan:
1. In MetaMask click on the transaction to open it then click `Copy Transaction ID`. Open [ObscuroScan](http://testnet-obscuroscan.uksouth.azurecontainer.io/).
1. Paste your copied transaction ID into the search box to find your individual guess transaction. Note how the transaction data visible to everyone is encrypted in the field `EncryptedTxBlob` and how (for now) the transaction is decrypted to allow developers to confirm correct behaviour of the network.
1. You can see your guess as the number at the right hand end of the `input` value in a hexadecimal format, e.g. a guess of 99 is shown as 63.

### Known Issues and Limitations
1. You requested game tokens using the Discord thread but there was no response.
    * **Cause**: this is a manual process for the moment.
    * **Workaround**: please be patient. The game tokens will be sent to your account as soon as possible.
1. When making a guess nothing appears to happen. I win neither the prize pool nor does the prize pool increase. MetaMask shows a failed transaction in the `Activity` tab for the guess submission.
    * **Cause 1**: the guessing game has been reset between this and your previous guess. As a result your nonce is out of sequence. 
    * **Workaround**: reset your MetaMask account data in MetaMask using the option `Settings > Advanced > Reset Account` 
    * **Cause 2**: your account does not have any game tokens available.
    * **Workaround**: request game tokens by posting a message with your wallet address to [this Discord thread](https://discord.com/channels/916052669955727371/1004391962733989960).
1. There is no confirmation you have made a correct guess.
    * **Cause**: the UI does not currently provide confirmation feedback when you make a correct guess.
    * **Workaround**: look at the value of the prize pool. When you correctly guess the secret number the prize pool will reduce to zero.

### Next Steps

Now you have enjoyed playing the guessing game you are invited to make it even better! Go ahead and fork the guessing game code in this [GitHub repository](https://github.com/obscuronet/number-guessing-game) and bring your own contributions to Obscuro Testnet.
Of course, you are free to [deploy any smart contract](https://docs.obscu.ro/testnet/deploying-a-smart-contract.html) to the testnet.   