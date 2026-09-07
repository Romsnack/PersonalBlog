---
title: Votre conteneur a supprimé le secret. La couche l'a gardé.
date: 2026-08-20
tags: [containers, docker, security]
summary: Ce qu'est réellement une image sur le disque, pourquoi un RUN rm ne supprime jamais rien, et trois façons d'extraire des secrets d'une couche qu'un conteneur en cours d'exécution jure pourtant vide.
translationKey: container-layers-secrets
slug: votre-conteneur-a-supprime-le-secret
---

Une image de conteneur n'est ni une machine ni un système de fichiers. C'est
une pile de tarballs et un fichier JSON qui décrit l'ordre dans lequel les
décompresser. Une fois cette phrase admise, toute une catégorie de « mais
comment cette clé a-t-elle bien pu atterrir sur GitHub ? » cesse d'être un
mystère.

En résumé : **une couche ne s'écrit qu'en ajout, jamais en modification**.
Supprimer un fichier revient à écrire une nouvelle couche qui dit « ignore ce
fichier », pendant que la couche contenant les octets d'origine reste
exactement où elle était, livrée à tous ceux qui font un `pull` de l'image.

## Un build qui a l'air correct

Voici un Dockerfile qui contient trois erreurs. Toutes les trois sont
courantes, et aucune n'est visible dans le conteneur final.

```dockerfile
FROM alpine:3.20

# Tout le contexte de build est copié, .env compris.
COPY . /app
WORKDIR /app

# Le « correctif » : le supprimer dans une couche ultérieure.
RUN rm -f /app/.env

# Un secret passé en argument de build.
ARG NPM_TOKEN
RUN echo "//registry.npmjs.org/:_authToken=${NPM_TOKEN}" > /root/.npmrc \
 && rm -f /root/.npmrc

CMD ["sh"]
```

Le `.env` posé à côté est on ne peut plus banal :

```ini
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

Construisez, lancez, et allez voir. Le conteneur est impeccable :

```console
$ docker run --rm leaky-demo:1 sh -c 'ls -a /app; cat /app/.env; cat /root/.npmrc'
.
..
Dockerfile
cat: can't open '/app/.env': No such file or directory
cat: can't open '/root/.npmrc': No such file or directory
```

Les deux fichiers ont disparu. On peut livrer, non ?

## Comment les couches sont réellement construites

Chaque instruction qui modifie le système de fichiers produit une couche : une
archive tar de *seulement ce qui a changé* pendant cette étape. Le runtime les
empile avec un système de fichiers union, si bien que le conteneur voit le
résultat fusionné — les couches supérieures masquant les inférieures.

C'est dans ce masquage que tout se joue. Quand un `RUN rm` supprime un fichier, le
système de fichiers union ne peut pas descendre dans une couche inférieure pour
la modifier, parce que les couches inférieures sont immuables et partagées
entre images par empreinte. Il fait donc la seule chose possible : il écrit un
**fichier whiteout** dans la nouvelle couche. Un marqueur vide nommé
`.wh.<nomdufichier>` qui signifie « au moment de fusionner la pile, fais comme
si ce qu'il y a en dessous de moi n'existait pas ».

Les octets en dessous sont intacts. Ils sont simplement recouverts.

`docker history` montre les couches, de la plus récente à la plus ancienne — à
lire de bas en haut :

```console
$ docker history leaky-demo:1
IMAGE          CREATED CREATED BY                                      SIZE
4c8b70fe6c4d   ...     /bin/sh -c #(nop)  CMD ["sh"]                   0B
6734cc35b915   ...     |1 NPM_TOKEN=npm_S3cr3tT0k3nExample012345678…   0B
da79965154e7   ...     /bin/sh -c #(nop)  ARG NPM_TOKEN                0B
92d0e72f7de6   ...     /bin/sh -c rm -f /app/.env                      0B
e545a7f805f5   ...     /bin/sh -c #(nop) WORKDIR /app                  0B
9c25a27592af   ...     /bin/sh -c #(nop) COPY dir:e944c2b43443471d0…   8.19kB
d9e853e87e55   4 months ago  CMD ["/bin/sh"]                           0B
<missing>      4 months ago  ADD alpine-minirootfs-3.20.10-x86_64.…    9.44MB
```

Remarquez que la couche `rm -f /app/.env` coûte **0B**. Elle n'a pas récupéré
les 8,19 ko ajoutés par le `COPY`. Supprimer un fichier dans une image de
conteneur rend l'image *plus grosse*, jamais plus petite.

## Attaque 1 : l'argument de build est déjà visible

Regardez à nouveau cette deuxième ligne. La troncature la cache, alors demandez
la commande complète :

```console
$ docker history leaky-demo:1 --no-trunc --format '{{.CreatedBy}}'
```

```text
|1 NPM_TOKEN=npm_S3cr3tT0k3nExample0123456789 /bin/sh -c echo
"//registry.npmjs.org/:_authToken=${NPM_TOKEN}" > /root/.npmrc && rm -f /root/.npmrc
```

Le voilà, en clair, dans des métadonnées livrées avec l'image. Rien à
décompresser, et `docker history` fonctionne sur une image récupérée depuis un
registre. **Un argument de build n'est pas un secret.** C'est un commentaire
collé à votre image, valeur comprise.

C'est le contrôle le moins coûteux qui soit — une seule commande — et il a sa
place dans votre pipeline, que vous fassiez ou non le reste de cet article.

## Attaque 2 : docker save et un peu de fouille

`docker history` ne voit que les métadonnées. Pour le contenu des fichiers, il
faut les couches elles-mêmes, et `docker save` envoie l'image entière sur la
sortie standard, sous forme de tar :

```console
$ mkdir unpacked && docker save leaky-demo:1 | tar -x -C unpacked
$ ls unpacked
blobs  index.json  manifest.json  oci-layout
```

`manifest.json` nomme le blob de configuration et liste les couches dans leur
ordre d'empilement. Chaque entrée sous `blobs/sha256/` est soit des métadonnées
JSON, soit une tarball de couche compressée en gzip.

Et là, le piège. Le réflexe évident échoue :

```console
$ grep -rl 'AKIA' unpacked/blobs/sha256/
$
```

Rien. Non pas parce que la clé est absente, mais parce que les couches sont
**gzippées** — `grep` scanne des octets compressés. Ce faux négatif est à
garder en tête, car c'est exactement le résultat qui pousse à déclarer une image
saine. Il faut d'abord décompresser :

```console
$ for f in unpacked/blobs/sha256/*; do
    file -b --mime-type "$f" | grep -q gzip || continue
    gzip -dc "$f" | grep -qa 'AKIA' && echo "HIT: $(basename $f)"
  done
HIT: 862c774e0067e3fae7ab9dd17fdbeb64f5e888ab8f0f0b97545cd60be8fec9b7
```

Comparez le contenu de cette couche avec celui de la couche empilée juste
au-dessus, et toute l'histoire est là :

```console
$ gzip -dc unpacked/blobs/sha256/862c774e0067* | tar -t
app/
app/.env
app/Dockerfile

$ gzip -dc unpacked/blobs/sha256/e5019b483794* | tar -t
app/
app/.wh..env
```

La deuxième couche porte `app/.env`. La troisième porte `app/.wh..env` — le
whiteout, le marqueur « fais comme s'il n'était plus là », et rien d'autre. Le
`rm` a posé un marqueur de suppression ; il n'a rien fait disparaître.

Lisez le fichier directement depuis le tar, sans le moindre conteneur :

```console
$ gzip -dc unpacked/blobs/sha256/862c774e0067* | tar -xO app/.env
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

Récupération complète, à partir de l'image qui affichait un répertoire vide
deux commandes après son démarrage.

## Attaque 3 : laisser dive faire le travail

La méthode manuelle vaut le coup d'être faite une fois, pour savoir ce que
l'outillage fait à votre place. Ensuite, utilisez
[dive](https://github.com/wagoodman/dive). Il reconstruit le système de
fichiers à chaque couche et vous montre ce que chaque étape a ajouté, modifié
et supprimé. En interactif :

```sh
dive leaky-demo:1
```

Mais le mode intéressant pour un pipeline est `--ci`, non interactif et qui
renvoie un code de sortie :

```console
$ dive leaky-demo:1 --ci
Analyzing image...
  efficiency: 99.9987 %
  wastedBytes: 102 bytes (102 B)
  userWastedPercent: 20.5231 %
Inefficient Files:
Count  Wasted Space  File Path
    2         102 B  /app/.env
Result:FAIL [Total:3] [Passed:1] [Failed:1] [Skipped:1]
```

Le `Count 2` est l'indice : `/app/.env` a été touché dans deux couches
différentes — ajouté dans l'une, recouvert d'un whiteout dans l'autre. dive
mesure l'espace gaspillé, il ne chasse pas les secrets, mais **« ce fichier
existe dans une couche et disparaît dans une couche ultérieure » est très
exactement la signature d'un secret qui fuit**, ce qui fait d'un outil
d'efficacité un détecteur étonnamment efficace.

Si vous ne voulez pas installer dive en local, il s'exécute depuis sa propre
image :

```sh
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  wagoodman/dive:latest leaky-demo:1 --ci
```

Docker en mode rootless place sa socket dans `/run/user/$(id -u)/docker.sock` —
montez ce chemin-là, sinon dive vous dira que le démon ne tourne pas alors
qu'il tourne.

## Comment corriger

Trois erreurs, trois correctifs, et aucun n'est un meilleur `rm`.

**Gardez le fichier hors du contexte.** `COPY . /app` copie tout ce qui se
trouve dans le répertoire, y compris les fichiers que vous aviez oubliés. Un
`.dockerignore` les arrête à la porte :

```text
.env
.git
```

`.git` compte autant que `.env` : il transporte tous les secrets que quelqu'un
a un jour commités avant de faire marche arrière.

**Montez les secrets de build au lieu de les passer.** BuildKit expose un
fichier sous `/run/secrets/` le temps d'un seul `RUN` et ne l'enregistre jamais
dans une couche ni dans l'historique :

```dockerfile
RUN --mount=type=secret,id=npm_token \
    npm config set //registry.npmjs.org/:_authToken="$(cat /run/secrets/npm_token)" \
 && npm ci
```

```sh
docker build --secret id=npm_token,src=token.txt -t app:1 .
```

**Utilisez des builds multi-étapes.** Faites le travail qui réclame des secrets
dans une étape de build dédiée, puis `COPY --from=builder` uniquement
l'artefact. Les couches dont vous ne copiez rien n'atteignent jamais l'image
finale.

Une fois l'image reconstruite avec les trois correctifs, la même analyse revient
vide — aucun `.env` dans la moindre couche, et l'historique enregistre le chemin
de montage plutôt que le jeton :

```console
$ docker history fixed-demo:2 --no-trunc --format '{{.CreatedBy}}' | head -2
CMD ["sh"]
RUN /bin/sh -c echo "//registry.npmjs.org/:_authToken=$(cat /run/secrets/npm_token)" ... # buildkit
```

C'est le `$(cat ...)` littéral qui a été stocké. La valeur n'a jamais existé
ailleurs que dans la mémoire de cette unique commande.

## Ce que tout le monde oublie

Si un secret a un jour atteint une couche publiée, **reconstruire ne corrige
rien.** Quiconque a récupéré ce tag a toujours les octets, et les registres
conservent les blobs adressés par empreinte même après le déplacement d'un tag.
Réécrire l'image, c'est du nettoyage, pas de la remédiation.

Effectuez la rotation du secret. Reconstruisez seulement ensuite.

Voilà tout : une image ne s'écrit qu'en ajout, `rm` ne fait que poser un
marqueur, et trois commandes suffisent à savoir ce que vous avez réellement
livré.
