---
title: Votre lockfile ne sait pas de quel registre vient un nom
date: 2026-10-26
tags: [supply-chain, npm, security]
summary: Comment un gestionnaire de paquets résout réellement un nom, pourquoi un nom de paquet interne jamais réservé côté public reste un nom public comme un autre, et une reproduction complète avec deux vrais registres et un seul nom de paquet — où la version la plus haute l'emporte, sans rien dans les paquets à part une ligne qui annonce d'où ils viennent.
translationKey: dependency-confusion
slug: confusion-de-dependances
---

En février 2021, le chercheur en sécurité Alex Birsan a publié [un article](https://medium.com/@alex.birsan/dependency-confusion-4a5d60fec610)
où il raconte comment il a fait tourner du code dans les réseaux d'Apple, de
Microsoft, de Tesla, d'Uber et de dizaines d'autres entreprises. Pas en
exploitant une faille dans leur code : en publiant, sur les registres publics
npm, PyPI et RubyGems, des paquets portant les bons *noms*. Certains de ces
noms internes avaient fuité — des `package.json`, des messages Slack, des
logs d'erreur traînant quelque part sur l'internet ouvert — et Birsan n'a
jamais eu besoin de toucher le moindre système interne : les gestionnaires de
paquets ont fait tout le travail à sa place, parce que c'est exactement ce
pour quoi ils sont conçus. Trouver la version la plus récente d'un nom, peu
importe où elle se trouve.

Et c'est toute l'attaque. Pas d'exploit, pas de malware à livrer, pas de
phishing. Une simple collision de noms, et un resolver qui fait son travail
correctement.

## Ce que fait vraiment la résolution

Un gestionnaire de paquets ignore ce que « interne » veut dire. Écrivez
`@acme/build-tools` dans un `package.json`, et npm n'a qu'une chose à faire :
trouver quelque chose qui répond à ce nom et à cette plage de version, puis
installer le plus récent. Par défaut, il n'interroge qu'un seul registre —
celui configuré — et ce registre ne demande jamais qui vous êtes. Tant qu'une
entreprise n'a pas explicitement dit au resolver *ce nom-là ne vit qu'ici*,
celui-ci l'acceptera de n'importe où — la première réponse venue, ou celle
dont le numéro de version est le plus haut.

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

Enlevez cette règle, et il ne reste qu'un seul registre configuré — qui
répondra pour *n'importe quel* nom qu'on lui demande, y compris celui que
vous pensiez garder privé.

## La reproduire, sans rien mettre de dangereux dans les paquets

Plutôt que de décrire le mécanisme, autant le reproduire. Voici une
démonstration complète avec deux registres jetables, tournant sur ma propre
machine : [Gitea](https://about.gitea.com/) et son registre de paquets npm
intégré, à la place d'un registre interne d'entreprise, et une instance
[Verdaccio](https://verdaccio.org/) locale à la place du registre npm public
— rien ici ne touche le vrai npm. Les deux paquets ne font qu'une chose à
l'installation : afficher une ligne disant de quel registre ils viennent.
Aucun accès fichier, aucun appel réseau, aucune charge utile. L'intérêt est
dans la *résolution de nom*, pas dans ce qu'un paquet malveillant pourrait
faire une fois qu'il a gagné — ça, ce sera pour un autre article.

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

**Le paquet « squatté »**, publié sur le registre public sous exactement le
même nom, avec juste un numéro de version plus élevé — c'est la seule chose
qui fait toute la différence :

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
registre, sans la moindre règle de scope pour `@acme`. C'est exactement à
quoi ressemble un `.npmrc` par défaut, celui que personne n'a jamais touché.

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

Cette ligne `postinstall` n'a rien d'anecdotique. npm, comme la plupart des
gestionnaires de paquets, exécute automatiquement des scripts de cycle de vie
à l'installation, avec les droits de celui qui a lancé la commande. En CI,
c'est en général un compte de service qui a accès au code source, aux
secrets et à un chemin de déploiement. Remplacez le `console.log` par
n'importe quoi d'autre, et la « démo » devient l'incident : cette ligne fait
tout le travail qu'ont fait les vrais paquets de Birsan.

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

Même projet, même plage `"*"`, les deux registres hébergent toujours chacun
un `@acme/build-tools` — mais avec la règle de scope, le registre public
n'est même plus interrogé pour ce nom. Rien à gagner d'un côté : il n'y a
tout simplement plus de compétition.

Le principe se transpose à tous les écosystèmes, mais le mécanisme change
d'un gestionnaire à l'autre, et les pièges avec :

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
  `npm install`, en CI), avec un registre qui vérifie les hachages
  d'intégrité. Un lockfile fixe une version *et* le hachage des octets
  exacts résolus la dernière fois : une substitution même nom / même version
  échouera à cette vérification. Ça ne protège en revanche en rien la toute
  première installation, celle qui crée le lockfile — raison de plus pour
  que la règle de scope ou l'index unique existe en parallèle.

## Ce que les gens zappent

Réserver les noms sur le registre public, tout le monde en parle, presque
personne ne le fait vraiment. Il faudrait qu'une entreprise énumère tous les
noms de paquets internes qu'elle a jamais utilisés — y compris ceux dont plus
personne ne se souvient, hérités d'un projet livré une fois et jamais
retouché depuis. Et ce sont justement ces noms-là, les plus susceptibles de
traîner encore dans un vieux `package-lock.json`, versionné dans un fork
public de quelque chose, qui n'attendent qu'à être repris.

Corriger la config ne fait pas disparaître le problème de fond : un nom de
paquet n'est « interne » que dans la configuration de votre propre
outillage. Pour n'importe quel registre, et pour quiconque sait lire cette
configuration, ce n'est jamais qu'une chaîne de caractères — et une chaîne de
caractères, tout le monde peut la réserver.
