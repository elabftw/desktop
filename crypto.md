# Encryption workflow

## Création du profil
1. L'utilisateur choisit une passphrase.
2. L'app génère un salt aléatoire.
3. L'app dérive une clé symétrique avec Argon2id(passphrase, salt).
4. L'app chiffre un petit texte fixe de vérification avec cette clé.
5. L'app stocke dans index.json :
    - uuid
    - display_name
    - created_at
    - salt
    - encrypted_verifier

## Déverrouillage du profil
1. L'utilisateur saisit sa passphrase.
2. L'app relit le salt depuis index.json.
3. L'app redérive la même clé avec Argon2id(passphrase, salt).
4. L'app tente de déchiffrer encrypted_verifier.
5. Si ça échoue : mauvaise passphrase.
6. Si ça réussit : l'app garde la clé en mémoire dans activeKey.


## Chiffrement des entrées
1. Quand l'utilisateur sauvegarde une entrée, title/body sont chiffrés avec activeKey.
2. La DB SQLite stocke seulement les ciphertexts.
3. Quand l'utilisateur relit une entrée, title/body sont déchiffrés avec activeKey.
4. Au logout, activeKey est effacée de la mémoire.

En résumé, la logique repose sur `passphrase forte + Argon2id + salt unique + XChaCha20-Poly1305`
