---
title: Votre lockfile ne sait pas de quel registre vient un nom
date: 2026-10-26
tags: [supply-chain, npm, security]
summary: Comment un gestionnaire de paquets résout réellement un nom en paquet, pourquoi un nom de paquet interne sans équivalent réservé sur l'index public est un nom public, et une reproduction complète — deux vrais registres, un seul nom de paquet, la version la plus haute gagne — sans rien dans aucun des deux paquets à part une ligne qui dit d'où il vient.
translationKey: dependency-confusion
slug: confusion-de-dependances
---

En février 2021, le chercheur en sécurité Alex Birsan a publié [un article](https://medium.com/@alex.birsan/dependency-confusion-4a5d60fec610)
décrivant comment il a fait exécuter du code à l'intérieur des réseaux
d'Apple, Microsoft, Tesla, Uber et de dizaines d'autres entreprises — non pas
en exploitant une vulnérabilité dans leur code, mais en publiant des paquets
portant les bons *noms* sur les registres publics npm, PyPI et RubyGems.
Certains de ces noms de paquets internes avaient fuité depuis des fichiers
`package.json`, des messages Slack et des logs d'erreur récupérés sur
l'internet ouvert. Il n'a jamais touché le moindre système interne. Les
gestionnaires de paquets ont fait le reste tout seuls, parce que c'est
exactement ce pour quoi ils sont conçus : trouver la version la plus récente
d'un nom, où qu'elle se trouve.

C'est toute l'attaque. Pas d'exploit, pas de mécanisme de livraison de
malware, pas de phishing. Juste une collision de noms et un resolver qui fait
correctement son travail.

## Ce que fait réellement la résolution

Un gestionnaire de paquets ne sait pas ce que « interne » veut dire. Quand
vous écrivez `@acme/build-tools` dans un `package.json`, le seul travail de
npm est : trouver quelque chose qui répond à ce nom, correspondant à cette
plage de version, et installer la correspondance la plus récente. Par
défaut, il interroge un seul registre — celui qui est configuré — et ce
registre ne vérifie pas qui vous êtes. Si une entreprise ne dit jamais
explicitement au resolver *ce nom ne vit qu'ici*, le resolver l'acceptera
sans problème depuis n'importe où, que ce soit la première réponse ou celle
qui gagne sur le numéro de version.

<figure class="diagram">
<svg viewBox="0 0 720 220" role="img" aria-labelledby="fig1t">
  <title id="fig1t">Le flux prévu : une règle de scope envoie le nom de paquet interne spécifiquement vers le registre interne, en contournant entièrement le registre public.</title>
  <defs><marker id="b1" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8" y="16" width="160" height="40" rx="4"/>
  <text class="d-t d-b" x="88" y="40" text-anchor="middle">npm install</text>
  <rect class="d-box-a" x="280" y="16" width="200" height="40" rx="4"/>
  <text class="d-a d-b" x="380" y="40" text-anchor="middle">@acme:registry</text>
  <rect class="d-box" x="560" y="16" width="152" height="40" rx="4"/>
  <text class="d-t d-b" x="636" y="40" text-anchor="middle">index public</text>
  <g class="d-arrow-a" marker-end="url(#b1)">
    <path d="M168,36 H275"/>
  </g>
  <text class="d-m" x="222" y="28" text-anchor="middle">@acme/build-tools</text>
  <g class="d-dash">
    <path d="M480,36 H555"/>
  </g>
  <text class="d-m" x="518" y="28" text-anchor="middle">tout le reste</text>
  <rect class="d-box-a" x="280" y="110" width="200" height="60" rx="4"/>
  <text class="d-a d-b" x="380" y="134" text-anchor="middle">registre interne</text>
  <text class="d-m" x="380" y="152" text-anchor="middle">@acme/build-tools@1.2.0</text>
  <rect class="d-box" x="560" y="110" width="152" height="60" rx="4"/>
  <text class="d-t d-b" x="636" y="134" text-anchor="middle">index public</text>
  <text class="d-m" x="636" y="152" text-anchor="middle">@acme/build-tools@99.0.0</text>
  <g class="d-arrow-a" marker-end="url(#b1)">
    <path d="M380,56 V105"/>
  </g>
  <g class="d-dash">
    <path d="M636,56 V105"/>
  </g>
  <text class="d-m" x="380" y="196" text-anchor="middle">installé — seul candidat interrogé</text>
  <text class="d-m" x="636" y="196" text-anchor="middle">jamais interrogé pour ce nom</text>
</svg>
<figcaption>Avec une règle de scope en place, le resolver ne demande même jamais à l'index public pour <code>@acme/*</code> — il n'y a pas de course à gagner puisqu'il n'y a pas de compétition.</figcaption>
</figure>

Sans cette règle, il n'y a qu'un seul registre configuré, et il répondra pour
*n'importe quel* nom qu'on lui demande — y compris celui que vous pensiez
garder privé.

## La reproduire, sans rien de dangereux dans aucun des deux paquets

Pour montrer le mécanisme plutôt que de simplement le décrire, voici une
reproduction complète utilisant deux registres jetables sur ma propre
machine : [Gitea](https://about.gitea.com/) avec son registre de paquets npm
intégré (à la place du registre interne d'une entreprise) et une instance
[Verdaccio](https://verdaccio.org/) locale (à la place du registre npm
public — pour que rien dans cette démo ne touche jamais le vrai npm). Les
deux paquets font exactement une chose à l'installation : afficher une ligne
disant de quel registre ils viennent. Aucun accès fichier, aucun appel
réseau, aucune charge utile d'aucune sorte — tout l'intérêt est la
*résolution de nom*, pas ce qu'un paquet malveillant pourrait faire une fois
qu'il a gagné, ce qui est un autre article.

**Le paquet interne « légitime »**, publié sur Gitea en version `1.2.0` :

```json
{
  "name": "@acme/build-tools",
  "version": "1.2.0",
  "scripts": { "postinstall": "node ./announce.js" }
}
```

```js
// announce.js
console.log("[@acme/build-tools@1.2.0] installed from the INTERNAL registry (Gitea).");
```

```console
$ npm publish
npm notice 📦  @acme/build-tools@1.2.0
npm notice Publishing to http://localhost:3000/api/packages/acme/npm/ with tag latest and default access
+ @acme/build-tools@1.2.0
```

**Le paquet « squatté »**, publié sur le substitut public sous exactement le
même nom, avec un numéro de version plus élevé — la seule information qui
décide de tout :

```json
{
  "name": "@acme/build-tools",
  "version": "99.0.0",
  "scripts": { "postinstall": "node ./announce.js" }
}
```

```js
// announce.js
console.log("[@acme/build-tools@99.0.0] installed from the PUBLIC registry (attacker-controlled) — THIS IS THE CONFUSION.");
```

```console
$ npm publish
npm notice 📦  @acme/build-tools@99.0.0
npm notice Publishing to http://localhost:4873/ with tag latest and default access
+ @acme/build-tools@99.0.0
```

Passons au côté vulnérable : un projet dont le `.npmrc` pointe vers un seul
registre, sans aucune règle de scope pour `@acme` — ce qui correspond
exactement à l'apparence d'un `.npmrc` par défaut, jamais configuré.

```ini
# .npmrc
registry=http://localhost:4873/
```

```json
{
  "dependencies": { "@acme/build-tools": "*" }
}
```

```console
$ npm install --foreground-scripts

> @acme/build-tools@99.0.0 postinstall
> node ./announce.js

[@acme/build-tools@99.0.0] installed from the PUBLIC registry (attacker-controlled) — THIS IS THE CONFUSION.

added 1 package, and audited 2 packages in 468ms
```

<figure class="diagram">
<svg viewBox="0 0 720 196" role="img" aria-labelledby="fig2t">
  <title id="fig2t">Sans règle de scope, le seul registre configuré répond aussi pour le nom interne, et la version la plus élevée gagne quel que soit le côté qui l'a publiée.</title>
  <defs><marker id="b2" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8" y="16" width="160" height="40" rx="4"/>
  <text class="d-t d-b" x="88" y="40" text-anchor="middle">npm install</text>
  <text class="d-m" x="88" y="66" text-anchor="middle">"@acme/build-tools": "*"</text>
  <rect class="d-box" x="280" y="16" width="200" height="40" rx="4"/>
  <text class="d-t d-b" x="380" y="40" text-anchor="middle">registry=…</text>
  <text class="d-m" x="380" y="66" text-anchor="middle">un seul registre, pas de règle</text>
  <g class="d-arrow" marker-end="url(#b2)">
    <path d="M168,36 H275"/>
  </g>
  <rect class="d-box-a" x="560" y="90" width="152" height="52" rx="4"/>
  <text class="d-a d-b" x="636" y="112" text-anchor="middle">index public</text>
  <text class="d-m" x="636" y="128" text-anchor="middle">99.0.0 — la plus haute</text>
  <g class="d-arrow-a" marker-end="url(#b2)">
    <path d="M380,56 V80 Q380,96 440,100 T556,110"/>
  </g>
  <rect class="d-box" x="280" y="90" width="200" height="52" rx="4" opacity="0.5"/>
  <text class="d-m" x="380" y="112" text-anchor="middle">registre interne 1.2.0</text>
  <text class="d-m" x="380" y="128" text-anchor="middle">jamais interrogé</text>
  <text class="d-m" x="636" y="168" text-anchor="middle">installé — le postinstall tourne avec cet utilisateur</text>
</svg>
<figcaption>« * » signifie « n'importe quelle version » pour le resolver, et le seul candidat qu'il voit jamais est ce que renvoie cet unique registre. Le caractère interne du nom est une convention dont le resolver n'a jamais été informé.</figcaption>
</figure>

Cette ligne `postinstall` n'est pas cosmétique. npm, comme la plupart des
gestionnaires de paquets, exécute automatiquement des scripts de cycle de
vie à l'installation, avec les permissions de celui qui a lancé
l'installation — ce qui, dans un pipeline CI, est généralement un compte de
service ayant accès au code source, aux secrets et à un chemin de
déploiement. Remplacez le `console.log` par n'importe quoi d'autre et la
« démo » devient l'incident ; cette ligne fait tout le travail que faisaient
les vrais paquets de Birsan.

## Le correctif : dire au resolver ce qu'on ne lui avait jamais dit

```ini
# .npmrc
registry=http://localhost:4873/
@acme:registry=http://localhost:3000/api/packages/acme/npm/
```

```console
$ npm install

$ cat node_modules/@acme/build-tools/package.json | grep version
  "version": "1.2.0",
```

Même projet, même plage `"*"`, mêmes deux registres contenant toujours un
paquet appelé `@acme/build-tools` — mais la règle de scope fait que le
registre public n'est même jamais interrogé pour ce nom. Il n'y a pas de
course à perdre puisqu'il n'y a pas de course.

Cette unique ligne se généralise à travers les écosystèmes, même si le
mécanisme exact diffère et que les pièges sont différents eux aussi :

- **npm** — mappez explicitement chaque scope interne (`@votreorg:registry=…`),
  pour chaque `.npmrc` qui compte : postes des développeurs, CI, et toute
  image Docker de base qui lance `npm install`.
- **pip** — `--extra-index-url` est le piège classique : pip interroge *tous*
  les index configurés et, comme npm, prend la version la plus élevée
  trouvée sur l'ensemble d'entre eux — il ne privilégie pas le premier index
  listé. Utilisez `--index-url` pointant uniquement vers votre miroir
  interne, avec le registre public mis en proxy *derrière* lui (la plupart
  des registres internes — Artifactory, Nexus, le support d'index intégré de
  Gitea — savent le faire), plutôt qu'interrogé en parallèle.
- **Tout registre qui le permet** — réservez les noms internes sur le
  registre public aussi, même sous forme de paquets vides servant de
  placeholders. C'est la seule défense qui ne dépend pas du fait que tous les
  fichiers de config en aval soient corrects.
- **Partout** — committez le lockfile et installez depuis lui (`npm ci`, pas
  `npm install`, en CI) avec un registre qui vérifie les hachages
  d'intégrité. Un lockfile fixe une version *et* le hachage des octets
  exacts qu'il a résolus la dernière fois, donc une substitution même nom /
  même version sur un registre échoue malgré tout à la vérification de
  hachage — bien que cela ne protège en rien la première installation qui a
  créé le lockfile, ce qui explique pourquoi la règle de scope ou l'index
  unique doit exister indépendamment.

## Ce que les gens zappent

Réserver les noms sur le registre public fait l'objet de beaucoup de
discours et est rarement réellement fait, parce que cela suppose que
quelqu'un énumère tous les noms de paquets internes que votre entreprise a
jamais utilisés — y compris ceux dont plus personne ne se souvient, issus de
projets livrés une fois et jamais retouchés depuis. Ce sont exactement les
noms les plus susceptibles de se trouver encore dans un vieux
`package-lock.json` versionné dans un fork public de quelque chose, en
attente.

Le fait sous-jacent ne disparaît pas non plus en corrigeant la config : un
nom de paquet interne ne signifie « interne » qu'à l'intérieur de la
configuration de votre propre outillage. Pour n'importe quel registre, et
pour quiconque peut lire cette configuration, ce n'est qu'une chaîne de
caractères, et les chaînes de caractères sont libres d'être enregistrées.
