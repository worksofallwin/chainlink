// SPDX-License-Identifier: MIT
pragma solidity ^0.8.6;

/**
 * @notice A contract to handle access control of subscription management dependent on signing a Terms of Service
 */

interface ITermsOfServiceAllowList {
  /**
   * @notice Return the message data for the proof given to accept the Terms of Service
   * @param acceptor - The wallet address that has accepted the Terms of Service on the UI
   * @param recipient - The recipient address that the acceptor is taking responsibility for
   * @return Hash of the message data
   */
  function getMessageHash(address acceptor, address recipient) external pure returns (bytes32);

  /**
   * @notice Wrap a bytes32 message as an ethereum signed message
   * @param messageHash - Message hash produced from "getMessageHash"
   * @return Hash of the message data packed as "\x19Ethereum Signed Message\n" + len(msg) + msg
   */
  function getEthSignedMessageHash(bytes32 messageHash) external pure returns (bytes32);

  /**
   * @notice Check if the address is authorized for usage
   * @param sender The transaction sender's address
   * @return True or false
   */
  function isAllowedSender(address sender) external returns (bool);

  /**
   * @notice Check if the address is blocked for usage
   * @param sender The transaction sender's address
   * @return True or false
   */
  function isBlockedSender(address sender) external returns (bool);

  /**
   * @notice Allows access to the sender based on acceptance of the Terms of Service
   * @param acceptor - The wallet address that has accepted the Terms of Service on the UI
   * @param recipient - The recipient address that the acceptor is taking responsibility for
   * @param proof - Signed data produced by the Chainlink Functions Subscription UI
   */
  function acceptTermsOfService(address acceptor, address recipient, bytes calldata proof) external;

  /**
   * @notice Removes a sender's access if already authorized, and disallows re-acceptiong the Terms of Service
   * @param sender - Address of the sender to block
   */
  function blockSender(address sender) external;

  /**
   * @notice Re-allows a previosly blocked sender to accept the Terms of Service
   * @param sender - Address of the sender to unblock
   */
  function unblockSender(address sender) external;
}
