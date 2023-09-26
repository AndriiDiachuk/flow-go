import FlowServiceAccount from 0xFLOWSERVICEADDRESS
import FlowStorageFees from 0xFLOWSTORAGEFEESADDRESS
import FungibleToken from 0xFUNGIBLETOKENADDRESS
import FlowToken from 0xFLOWTOKENADDRESS

// This transaction sets up storage on any auth accounts that were created before the storage fees.
// This is used during bootstrapping a local environment
transaction() {

    prepare(
        service: auth(SaveValue, BorrowValue, Capabilities) &Account,
        fungibleToken: auth(SaveValue, Capabilities) &Account,
        flowToken: auth(SaveValue, Capabilities) &Account,
        feeContract: auth(SaveValue, Capabilities) &Account,
    ) {

        let authAccounts:[auth(SaveValue, Capabilities) &Account] = [service, fungibleToken, flowToken, feeContract]

        // Take all the funds from the service account.
        let tokenVault: auth(FungibleToken.Withdrawable) &FlowToken.Vault = service.storage
            .borrow<auth(FungibleToken.Withdrawable) &FlowToken.Vault>(from: /storage/flowTokenVault)
            ?? panic("Unable to borrow reference to the default token vault")

        for account in authAccounts {
            let storageReservation <- tokenVault.withdraw(amount: FlowStorageFees.minimumStorageReservation) as! @FlowToken.Vault

            let receiverCap = account.capabilities.get<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
            if receiverCap == nil || !receiverCap!.check() {
                FlowServiceAccount.initDefaultToken(account)
            }

            let receiver = account.capabilities.borrow<&{FungibleToken.Receiver}>(/public/flowTokenReceiver)
                ?? panic("Could not borrow receiver reference to the recipient's Vault")

            receiver.deposit(from: <-storageReservation)
        }
    }
}
