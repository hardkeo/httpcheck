httpcheck: Ferramenta de Verificação de Status HTTP
=================================================

`httpcheck` é uma ferramenta de linha de comando desenvolvida em Go para verificar o status HTTP de um URL. Ele foi projetado para ser simples e eficiente, tornando-o ideal para uso em verificações de saúde (health checks) em ambientes de produção, especialmente em containers.

Funcionalidades
--------------

* **Verificação de Status HTTP:** Verifica se o código de status da resposta HTTP corresponde aos códigos esperados.

* **Verificação do Corpo da Resposta:** Opcionalmente, verifica se o corpo da resposta contém uma string específica.

* **Configuração Flexível:**

    * URL e códigos de status podem ser fornecidos como argumentos de linha de comando ou através de variáveis de ambiente.

    * A string de pesquisa do corpo pode ser fornecida como argumento de linha de comando ou através de variável de ambiente.

* **Suporte a Containers:** Projetado para funcionar bem em containers, com suporte para configuração sem shell.

* **Opções de Configuração:**

    * Timeout de requisição configurável.

    * Suporte para métodos HTTP GET e HEAD.

    * Opção para ignorar erros de certificado TLS (para ambientes de teste).

    * Saída verbosa para depuração.

Uso
===

Sintaxe
------

```
./httpcheck -u  [-c ] [-t ] [-v] [-k] [-m ] [-b  | -B ]
./httpcheck -U  [-C ] [-t ] [-v] [-k] [-m ] [-b  | -B ]
```

Opções
------

* `-u`, `--url `: URL a ser verificada.

* `-U`, `--url-env-name `: Nome da variável de ambiente contendo o URL.

* `-c`, `--accepted-codes `: Lista de códigos HTTP esperados, separados por vírgula (ex: 200,404). Opcional se `-b` for usado.

* `-C`, `--accepted-codes-env-name `: Nome da variável de ambiente contendo os códigos aceitos. Opcional se `-b` for usado.

* `-t`, `--timeout `: Timeout da requisição HTTP em segundos (padrão: 5 segundos).

* `-k`, `--insecure`: Permite conexões TLS inseguras (ignora erros de certificado).

* `-v`, `--verbose`: Ativa o modo verbose.

* `-h`, `--help`: Exibe esta ajuda.

* `-m`, `--method `: Método HTTP a ser usado (padrão: GET).

* `-b`, `--body-contains `: String que o corpo da resposta deve conter (apenas para GET). Se usado, a verificação de códigos de status é opcional.

* `-B`, `--body-contains-env-name `: Nome da variável de ambiente contendo a string que o corpo da resposta deve conter (apenas para GET).

Casos de Uso
===========

Verificação Básica de Status
---------------------------

Verifica se um serviço está retornando o código de status 200:

```
./httpcheck -u http://localhost:8080/health
```

Verificação com Códigos de Status Aceitos Múltiplos
-------------------------------------------------

Verifica se o serviço retorna 200 ou 404:

```
./httpcheck -u http://localhost:8080/pagina_inexistente -c 200,404
```

Verificação com Timeout
----------------------

Verifica um serviço com um timeout de 10 segundos:

```
./httpcheck -u http://meu-servico-lento:8080/api -c 200 -t 10
```

Verificação com Variáveis de Ambiente
------------------------------------

Usando variáveis de ambiente para configurar a URL e os códigos aceitos:

```
export URL_TO_CHECK=http://localhost:8080/api/health
export ACCEPTED_STATUS_CODES="200,503"
./httpcheck -U URL_TO_CHECK -C ACCEPTED_STATUS_CODES
```

Verificação do Corpo da Resposta
-------------------------------

Verifica se o corpo da resposta contém a string "OK":

```
./httpcheck -u http://localhost:8080/status -b "OK"
```

Verificação do Corpo da Resposta com Variável de Ambiente
--------------------------------------------------------

```
export RESPONSE_BODY_CONTAINS="Funcionando"
./httpcheck -u http://localhost:8080/status -B RESPONSE_BODY_CONTAINS
```

Verificação com Método HEAD
--------------------------

Usando o método HEAD para verificar apenas os cabeçalhos:

```
./httpcheck -u http://localhost:8080/ -m HEAD -c 200
```

Uso em Contentores
=================

Com Shell
---------

Dockerfile:

```
ENV PORT=8080
ENV HTTP_ACCEPTED_CODES=200,404
HEALTHCHECK --interval=5s --timeout=3s CMD ["./httpcheck", "-u", "http://localhost:${PORT}/api/v1/health", "-c", "${HTTP_ACCEPTED_CODES}"]
```

Sem Shell
---------

Dockerfile:

```
ENV URL_TO_CHECK=http://localhost:8080/api/v1/health
ENV ACCEPTED_STATUS_CODES=200,404
HEALTHCHECK --interval=5s --timeout=3s CMD ["./httpcheck", "-U", "URL_TO_CHECK", "-C", "ACCEPTED_STATUS_CODES"]
```

Com verificação do corpo
-----------------------

Dockerfile:

```
ENV PORT=8080
ENV BODY_CHECK_STRING="OK"
HEALTHCHECK --interval=5s --timeout=3s CMD ["./httpcheck", "-u", "http://localhost:${PORT}/", "-b", "${BODY_CHECK_STRING}"]
```

No exemplo acima, o `httpcheck` irá verificar se o corpo da resposta contém a string "OK". Se contiver, o health check será considerado bem-sucedido.

Com verificação do corpo usando variável de ambiente
--------------------------------------------------

Dockerfile:

```
ENV URL_TO_CHECK=http://localhost:8080/
ENV RESPONSE_BODY_CONTAINS="OK"
HEALTHCHECK --interval=5s --timeout=3s CMD ["./httpcheck", "-U", "URL_TO_CHECK", "-B", "RESPONSE_BODY_CONTAINS"]
```

Neste caso, o `httpcheck` verifica se o corpo da resposta contém a string "OK", que é fornecida através da variável de ambiente `RESPONSE_BODY_CONTAINS`.

Contribuição
============

Contribuições são bem-vindas! Sinta-se à vontade para abrir issues e enviar pull requests.

Licença
=======

Este projeto está licenciado sob a [Licença MIT](https://opensource.org/licenses/MIT).
