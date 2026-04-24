# Cryptographic Signatures Explained Simply (Informative)

## What Signatures Are About
A cryptographic signature is digital proof that a message actually comes from the person or system named as sender, and that the content was not modified unnoticed during transit. In practical terms, anyone verifying a signed message can check whether sender and content match.

In a blockchain system, this proof is central. Validators, wallets, and applications continuously make decisions based on signed data. If this proof is no longer reliable, technical uncertainty quickly becomes an operational and trust problem.

Without signatures, any party could claim that a message came from someone else. With signatures, it can be verified whether a message is authorized. This protects not only individual transactions but also the shared reliability and trustworthiness of the network.

In practice, complex cryptographic methods are used for this. These methods are much more sophisticated than simple teaching examples and are specifically designed to make forgery practically infeasible.

## Asymmetric Signatures
In blockchain contexts, signatures are typically based on asymmetric cryptography. This means one party has two keys: a private key and a public key. The private key remains secret and is never shared. The public key is derived from the private key and can be distributed openly in the network.

The key point is that this derivation is designed not to be practically reversible. The private key should not be derivable from the public key. At the same time, the mathematical relation between both keys remains strong enough such that it allows signature verification to be computationally feasible.

## Alice/Bob Positive Case
Alice sends a message to Bob and signs it with her private key. The signature is transmitted together with the message. Bob can then verify whether message and signature match and whether the message corresponds to Alice's public key.

Intuitively, the signature can be viewed as a special check value derived from two parts: the message content and Alice's private key. This links identity and integrity. Identity is expressed by key binding, integrity by message binding. If either part changes, verification fails.

Bob verifies signature, message and Alice's public key together. If verification succeeds, Bob can assume the message came from Alice and was not altered since it was signed. This is the normal case: signature verification creates binding between sender, content, and recipient.

In the negative case, an attacker tries to position themselves between Alice and Bob and impersonate Alice. If signatures were no longer secure or could be forged, Bob might accept a manipulated message as authentic. The result would not just be an error in one message, but a fundamental loss of trust in network communication.

## Bridge to the PQ Topic
The core point of this RFC is not to explain specific algorithms in full detail, but to make the security assumptions behind real signature schemes explicit. Classical schemes are currently considered practically secure with typical parameters. With sufficiently capable quantum computers, this assumption may change for certain schemes.

This is exactly why Terra Classic components that rely on signature verification need a structured migration to new schemes: post-quantum schemes (PQ).
