# sigstore-attestor

## Overview
SIGSTORE is a project to enable signing containers using short-lived identities, such as OIDC tokens or SPIFFE IDs. This way organizations can sign their containers without having to have special "magic" keypairs on their buildhosts.

SIGSTORE works as follows:
 * The user logs into a special root CA, which generates a short-lived certificate
 * The root CA records this new short-lived certificate in a Certificate Transparency log
 * The user signs the container image with this keypair (using standard container signing methods -- Docker Content Trust/Notary)
 * In order to verify the signature, another user:
   * First checks the validity of the signature on the container in the normal way (Docker Content Trust/Notary)
   * Checks that the public key was really properly created by a logged-in user by checking the Certificate Transparency log

As you can see, SIGSTORE uses many of the same components as traditional PKI, but in a different way in order to create long-lived signatures based on short-lived credentials.

Almost all the work is performed by Sigstore's Cosign front-end.

In this demo, the public Docker Hub is used as the container registry, and the public instances of Fulcio (root CA) and Rekor (Certificate Transparency log) are used. In most real installations these would likely point to private servers.

## Use as a SPIRE Workload Attestor
The goal of this POC is to demonstrate that SIGSTORE signatures on local Docker containres can be used as the basis for a SPIRE workload attestor. That is, only a container signed by a specific person or build system can obtain a particular SPIFFE ID. 

## Directions to run
1. You'll need a Linux/Intel machine with GUI access. (Cosign launches a browser window to authenticate.)
1. In the env file in this directory, set DOCKER_USER to your Docker Hub username, and SIGSTORE_SUBJECT to the "Subject" name SIGSTORE will give you (typically your email address)
1. Run the script runall.sh to demonstrate creating an image, signing it, and attesting with SPIRE. You'll be presented with a login screen that lets you log into Sigstore with Google, Microsoft or Github credentials through federated identity midway through the process. 

## Questions
 *  In most SIGSTORE documentation, they suggest using an Kubernetes admission controller (or similar script on starting a container) to prevent unsigned containers from running at all. Is this redundant with that approach?

    * No, the two approaches can work together. First, a security admin might not trust their Kubernetes admins to install the admission controller, or configure it properly, so using SPIRE provides a second layer of defense. Second, an admission controller cannot control what a container does once it IS in the cluster; any signed container could impersonate any other signed container. SPIRE prevents this by only granting a specific signed identity to a specific container image.  

 * Does it make sense to call an external cosign binary rather than use Cosign's Go APIs?

    * At this point I believe it makes sense to use the external Cosign binary. Cosign is changing rapidly, and pulling a somewhat unstable and large code base directly into SPIRE seems unwise. Also, by calling an external binary, SPIRE will gracefully terminate the attestation process if something goes horribly wrong (for example if querying one of the external servers is very slow). Eventually, it should probably be integrated. 

 * There is an existing demo from Google of using SPIFFE identities to sign Cosign images. How does that compare to this POC?
    * The existing Google demo is about using the SPIFFE identity to log into Fulcio to grant a temporary certificate, rather than having to log in with human credentials. This POC is about using Cosign to validate the identity of containers at runtime, whether they are signed by a container or not. 

## TODO
 * Add unit tests and more documentation. 
 * Right now, we check the identity of a container by name. This is dangerous since the container locally may be different from the container in the registry, even if they have thes same name and tag. An additional step is needed to check that the hashes of both containers is the same (which will involve using the Docker Hub API).
  * Add more flexibility for different Fulcio and Rekor instances, and different registries (including local registries). Maybe even alternate container runtimes!