# Encryption workflow

## Création du profil
1. L'utilisateur choisit une passphrase.
2. L'app génère un UUID de profil.
3. L'app génère une paire de clés Ed25519 :
    - une clé publique ;
    - une clé privée.
4. L'app génère un salt aléatoire.
5. L'app dérive une clé symétrique avec `Argon2id(passphrase, salt)`.
6. L'app chiffre la clé privée Ed25519 avec cette clé symétrique.
7. L'app stocke dans `index.json` :
    - `uuid`
    - `display_name`
    - `created_at`
    - `public_key`
    - `salt`
    - `encrypted_private_key`

La passphrase n'est jamais stockée directement.

## Déverrouillage du profil

1. L'utilisateur saisit sa passphrase.
2. L'app relit depuis `index.json` :
    - `public_key`
    - `salt`
    - `encrypted_private_key`
3. L'app redérive la même clé symétrique avec `Argon2id(passphrase, salt)`.
4. L'app tente de déchiffrer `encrypted_private_key`.
5. Si le déchiffrement échoue : la passphrase est incorrecte.
6. Si le déchiffrement réussit, l'app vérifie que la clé privée déchiffrée correspond à la clé publique stockée.
7. Si la vérification réussit, l'app garde en mémoire :
    - `activeKey`, la clé symétrique dérivée de la passphrase ;
    - `activePrivateKey`, la clé privée Ed25519 déchiffrée.

La clé privée chiffrée sert donc aussi de vérification de passphrase : il n'y a pas de `encrypted_verifier` séparé.

## Chiffrement des entrées

1. Quand l'utilisateur sauvegarde une entrée, `title` et `body` sont chiffrés avec `activeKey`.
2. La DB SQLite stocke seulement les ciphertexts.
3. Quand l'utilisateur relit une entrée, `title` et `body` sont déchiffrés avec `activeKey`.
4. Au logout, `activeKey` et `activePrivateKey` sont effacées de la mémoire.

## Ce qui reste visible sur disque

Le fichier SQLite reste un fichier SQLite normal. Les métadonnées restent visibles, par exemple :

- les tables ;
- le schéma ;
- les IDs ;
- les timestamps ;
- le nombre d'entrées.

En revanche, le contenu sensible des entrées, comme `title` et `body`, est chiffré avant écriture.

## Résumé

La logique repose sur :

- une passphrase forte ;
- un salt unique par profil ;
- `Argon2id` pour dériver une clé symétrique depuis la passphrase ;
- `XChaCha20-Poly1305` pour chiffrer la clé privée Ed25519 et les entrées ;
- `Ed25519` pour donner une identité cryptographique persistante au profil.

En résumé :

```text
passphrase + salt
-> Argon2id
-> activeKey
-> decrypt encrypted_private_key
-> unlock profile
-> encrypt/decrypt SQLite entry title/body
