// SPDX-License-Identifier: Apache 2

pragma solidity >=0.7.0 <0.9.0;

import "./Structs.sol";

interface IMessageBus {
    // This method is called from contracts to publish messages to the other linked message bus.
    // nonce - This is provided and serves as deduplication nonce. It can also be used to group a batch of messages together.
    // topic - This is the topic for which the payload is published. 
    // payload - This is the actual message.
    // consistencyLevel - this is how many block confirmations to wait before publishing the message. 
    // Notice that consistencyLevel == 0 is still secure, but might make your protocol result more prone to reorganizations.
    // returns sequence - this is the unique id of the published message for the address calling the function. It can be used
    // to determine the order of incoming messages on the other side and if something is missing.
    function publishMessage(
        uint32 nonce,
        uint32 topic,
        bytes calldata payload, 
        uint8 consistencyLevel
    ) external returns (uint64 sequence);

    // This function verifies that a cross chain message provided by the caller has indeed been submitted from the other network
    // and returns true only if the challenge period for the message has passed.
    function verifyMessageFinalized(Structs.CrossChainMessage calldata crossChainMessage) external view returns (bool);
    
    // Returns the time when a message is final (when the rollup challenge period has passed). If the message was never submitted the call will revert.
    function getMessageTimeOfFinality(Structs.CrossChainMessage calldata crossChainMessage) external view returns (uint256);

    // This is the smart contract function which is used to store messages sent from the other linked layer. 
    // The function will be called by the ManagementContract on L1 and the enclave on L2. 
    // It should be access controlled and called according to the consistencyLevel and Obscuro platform rules.
    function submitOutOfNetworkMessage(Structs.CrossChainMessage calldata crossChainMessage, uint256 finalAfterTimestamp) external;

   /* function queryMessages(
        address      sender,
        bytes memory topic,
        uint256      fromIndex,
        uint256      toIndex
    ) external returns (bytes [] memory); */
}
