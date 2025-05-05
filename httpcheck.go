package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// httpcheck: Ferramenta de linha de comando para verificar o estado HTTP de um URL.
//
// Este programa realiza um pedido HTTP para o URL fornecido e, opcionalmente,
// verifica se o código de estado da resposta corresponde a um dos códigos HTTP
// aceites ou se o corpo da resposta contém uma string específica.
// Suporta a especificação do URL, códigos e string de pesquisa diretamente como
// argumentos ou através de variáveis de ambiente, sendo adequado para uso em
// contentores com e sem shell. Oferece opções para configuração de timeout,
// método HTTP, verificação do corpo da resposta, saída detalhada e para ignorar
// erros de certificado TLS (conexões inseguras).
//
// Autor: hardkeo@gmail.com
// Data de Criação: 2025-04-03
// Licença: MIT License
//
// Uso:
//   ./httpcheck -u <URL> [-c <códigos>] [-t <segundos>] [-v] [-k] [-m <GET|HEAD>] [-b <string> | -B <NOME_VAR_CORPO>]
//   ./httpcheck -U <NOME_VAR_URL> [-C <NOME_VAR_COD_ACEITOS>] [-t <segundos>] [-v] [-k] [-m <GET|HEAD>] [-b <string> | -B <NOME_VAR_CORPO>]
//
// Opções:
//   -u, --url <URL>:             URL a ser verificado.
//   -U, --url-env-name <NOME>:   Nome da variável de ambiente contendo o URL.
//   -c, --accepted-codes <códigos>: Lista de códigos HTTP esperados, separados por vírgula (ex: 200,404). Opcional se -b/--body-contains for usado.
//   -C, --accepted-codes-env-name <NOME>: Nome da variável de ambiente contendo os códigos aceites. Opcional se -b/--body-contains for usado.
//   -t, --timeout <segundos>:    Timeout do pedido HTTP em segundos (padrão: 5 segundos).
//   -k, --insecure:       Permite conexões TLS inseguras (ignora erros de certificado).
//   -v, --verbose:        Ativa o modo verbose.
//   -h, --help:           Exibe esta ajuda.
//   -m, --method <GET|HEAD>: Método HTTP a ser usado (padrão: GET).
//   -b, --body-contains <string>: String que o corpo da resposta deve conter (apenas para GET). Se usado, a verificação de códigos de estado é opcional.
//   -B, --body-contains-env-name <NOME>: Nome da variável de ambiente contendo a string que o corpo da resposta deve conter (apenas para GET).
//
// Exemplos de uso em Contentor:
//
// Com Shell (expansão de variáveis pelo shell):
//   Dockerfile:
//     ENV PORT=8080
//     ENV HTTP_ACCEPTED_CODES=200,404
//     ENV BODY_CHECK_STRING="OK"
//     HEALTHCHECK --interval=5s --timeout=3s CMD ["./httpcheck", "-u", "http://localhost:${PORT}/api/v1/health", "-c", "${HTTP_ACCEPTED_CODES}", "-b", "${BODY_CHECK_STRING}"]
//
// Sem Shell (leitura direta das variáveis pelo programa):
//   Dockerfile:
//     ENV URL_TO_CHECK=http://localhost:8080/api/v1/health
//     ENV ACCEPTED_STATUS_CODES=200,404
//     ENV RESPONSE_BODY_CONTAINS="OK"
//     HEALTHCHECK --interval=5s --timeout=3s CMD ["./httpcheck", "-U", "URL_TO_CHECK", "-C", "ACCEPTED_STATUS_CODES", "-B", "RESPONSE_BODY_CONTAINS"]
//
// Com timeout e método HEAD:
//   ./httpcheck -u http://localhost:8080/health -c 200 -t 10 -m HEAD
//
// Com verificação do corpo via variável de ambiente (método GET implícito), ignorando códigos:
//   ./httpcheck -u http://localhost:8080/status -B "RESPONSE_BODY_CONTAINS"
//
// Com verificação do corpo e códigos:
//   ./httpcheck -u http://localhost:8080/status -b "OK" -c 200

func main() {
	var url string
	var urlEnvName string
	var acceptedCodesStr string
	var acceptedCodesEnvName string
	var verbose bool
	var insecureSkipVerify bool
	var help bool
	var timeoutSeconds int = 5
	var httpMethod string = "GET"
	var bodyContains string
	var bodyContainsEnvName string // Nova variável para o nome da variável de ambiente do corpo

	args := os.Args[1:]

	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-u", "--url":
			if i+1 < len(args) {
				url = args[i+1]
				i++
			} else {
				fmt.Println("Erro: -u/--url precisa de um argumento.")
				os.Exit(1)
			}
		case "-U", "--url-env-name":
			if i+1 < len(args) {
				urlEnvName = args[i+1]
				i++
			} else {
				fmt.Println("Erro: -U/--url-env-name precisa de um argumento.")
				os.Exit(1)
			}
		case "-c", "--accepted-codes":
			if i+1 < len(args) {
				acceptedCodesStr = args[i+1]
				i++
			} else {
				fmt.Println("Erro: -c/--accepted-codes precisa de um argumento.")
				os.Exit(1)
			}
		case "-C", "--accepted-codes-env-name":
			if i+1 < len(args) {
				acceptedCodesEnvName = args[i+1]
				i++
			} else {
				fmt.Println("Erro: -C/--accepted-codes-env-name precisa de um argumento.")
				os.Exit(1)
			}
		case "-v", "--verbose":
			verbose = true
		case "-k", "--insecure":
			insecureSkipVerify = true
		case "-h", "--help":
			help = true
		case "-t", "--timeout":
			if i+1 < len(args) {
				t, err := strconv.Atoi(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Erro: -t/--timeout precisa de um inteiro válido: %v\n", err)
					os.Exit(1)
				}
				timeoutSeconds = t
				i++
			} else {
				fmt.Println("Erro: -t/--timeout precisa de um argumento.")
				os.Exit(1)
			}
		case "-m", "--method":
			if i+1 < len(args) {
				method := strings.ToUpper(args[i+1])
				if method == "GET" || method == "HEAD" {
					httpMethod = method
					i++
				} else {
					fmt.Fprintf(os.Stderr, "Erro: Método HTTP inválido: %s. Use GET ou HEAD.\n", method)
					os.Exit(1)
				}
			} else {
				fmt.Println("Erro: -m/--method precisa de um argumento.")
				os.Exit(1)
			}
		case "-b", "--body-contains":
			if i+1 < len(args) {
				bodyContains = args[i+1]
				i++
			} else {
				fmt.Println("Erro: -b/--body-contains precisa de um argumento.")
				os.Exit(1)
			}
		case "-B", "--body-contains-env-name": // Novo caso para -B
			if i+1 < len(args) {
				bodyContainsEnvName = args[i+1]
				i++
			} else {
				fmt.Println("Erro: -B/--body-contains-env-name precisa de um argumento.")
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Opção desconhecida: %s\n", arg)
			printHelp()
			os.Exit(1)
		}
	}

	if help {
		printHelp()
		os.Exit(0)
	}

	if urlEnvName != "" {
		url = os.Getenv(urlEnvName)
		if verbose {
			fmt.Printf("Usando URL da variável de ambiente: %s=%s\n", urlEnvName, url)
		}
	}

	if acceptedCodesEnvName != "" && bodyContainsEnvName == "" && bodyContains == "" {
		acceptedCodesStr = os.Getenv(acceptedCodesEnvName)
		if verbose {
			fmt.Printf("Usando códigos aceites da variável de ambiente: %s=%s\n", acceptedCodesEnvName, acceptedCodesStr)
		}
	}

	if bodyContainsEnvName != "" { // Nova lógica para obter bodyContains
		bodyContains = os.Getenv(bodyContainsEnvName)
		if verbose {
			fmt.Printf("Usando string de busca do corpo da variável de ambiente: %s=%s\n", bodyContainsEnvName, bodyContains)
		}
	}

	if url == "" {
		fmt.Println("Erro: O URL deve ser fornecido via argumentos ou variável de ambiente.")
		printHelp()
		os.Exit(1)
	}

	var acceptedCodeMap map[int]bool
	if acceptedCodesStr != "" {
		acceptedCodes := strings.Split(acceptedCodesStr, ",")
		acceptedCodeMap = make(map[int]bool)

		for _, codeStr := range acceptedCodes {
			code, err := strconv.Atoi(strings.TrimSpace(codeStr))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Erro ao converter código aceite '%s': %v\n", codeStr, err)
				os.Exit(1)
			}
			acceptedCodeMap[code] = true
		}
	}

	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}
	if insecureSkipVerify {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = transport
		if verbose {
			fmt.Println("Aviso: Conexões TLS inseguras estão habilitadas. Use com cautela.")
		}
	}

	var resp *http.Response
	var err error

	switch httpMethod {
	case "GET":
		resp, err = client.Get(url)
	case "HEAD":
		resp, err = client.Head(url)
	}

	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Erro ao fazer o pedido para %s (%s): %v\n", url, httpMethod, err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Verificar o corpo da resposta se --body-contains foi fornecido e o método é GET
	bodyCheckPassed := true;
	if bodyContains != "" && httpMethod == "GET" {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Erro ao ler o corpo da resposta: %v\n", err)
			}
			os.Exit(1)
		}
		body := string(bodyBytes)
		if !strings.Contains(body, bodyContains) {
			if verbose {
				fmt.Fprintf(os.Stderr, "Erro: O corpo da resposta não contém '%s'\n", bodyContains)
			}
			bodyCheckPassed = false
		} else if verbose {
			fmt.Printf("Sucesso: O corpo da resposta contém '%s'\n", bodyContains)
		}
	}

	// Verificar o código de estado se --body-contains NÃO foi usado ou a verificação do corpo passou
	if bodyContains == "" || bodyCheckPassed {
		if acceptedCodeMap != nil {
			if acceptedCodeMap[resp.StatusCode] {
				if verbose {
					fmt.Printf("Pedido bem-sucedido para %s (%s). Código de estado: %d (aceite)\n", url, httpMethod, resp.StatusCode)
				}
				os.Exit(0)
			} else {
				if verbose {
					fmt.Printf("Erro no pedido para %s (%s). Código de estado: %d (não aceite, esperado: %s)\n", url, httpMethod, resp.StatusCode, acceptedCodesStr)
				}
				os.Exit(1)
			}
		} else if bodyContains == "" {
			// Se nem códigos nem body-contains foram fornecidos, consideramos sucesso se o pedido não falhou
			if verbose {
				fmt.Printf("Pedido bem-sucedido para %s (%s). Código de estado: %d (nenhum código esperado)\n", url, httpMethod, resp.StatusCode)
			}
			os.Exit(0)
		} else if bodyCheckPassed {
			os.Exit(0) // Se body-contains foi usado e passou, e não há códigos, consideramos sucesso
		}
	} else {
		os.Exit(1) // Falha na verificação do corpo
	}
}

func printHelp() {
	fmt.Println("httpcheck: Ferramenta de linha de comando para verificar o estado HTTP de um URL.")
	fmt.Println()
	fmt.Println("Uso:")
	fmt.Println("  ./httpcheck -u <URL> [-c <códigos>] [-t <segundos>] [-v] [-k] [-m <GET|HEAD>] [-b <string> | -B <NOME_VAR_CORPO>]")
	fmt.Println("  ./httpcheck -U <NOME_VAR_URL> [-C <NOME_VAR_COD_ACEITOS>] [-t <segundos>] [-v] [-k] [-m <GET|HEAD>] [-b <string> | -B <NOME_VAR_CORPO>]")
	fmt.Println()
	fmt.Println("Opções:")
	fmt.Println("  -u, --url <URL>:             URL a ser verificado.")
	fmt.Println("  -U, --url-env-name <NOME>:   Nome da variável de ambiente contendo o URL.")
	fmt.Println("  -c, --accepted-codes <códigos>: Lista de códigos HTTP esperados, separados por vírgula (ex: 200,404). Opcional se -b/--body-contains for usado.")
	fmt.Println("  -C, --accepted-codes-env-name <NOME>: Nome da variável de ambiente contendo os códigos aceites. Opcional se -b/--body-contains for usado.")
	fmt.Println("  -t, --timeout <segundos>:    Timeout do pedido HTTP em segundos (padrão: 5 segundos).")
	fmt.Println("  -k, --insecure:       Permite conexões TLS inseguras (ignora erros de certificado).")
	fmt.Println("  -v, --verbose:        Ativa o modo verbose.")
	fmt.Println("  -h, --help:           Exibe esta ajuda.")
	fmt.Println("  -m, --method <GET|HEAD>: Método HTTP a ser usado (padrão: GET).")
	fmt.Println("  -b, --body-contains <string>: String que o corpo da resposta deve conter (apenas para GET). Se usado, a verificação de códigos de estado é opcional.")
	fmt.Println("  -B, --body-contains-env-name <NOME>: Nome da variável de ambiente contendo a string que o corpo da resposta deve conter (apenas para GET).")
	fmt.Println()
	fmt.Println("Exemplos de uso em Contentor:")
	fmt.Println("  Com Shell (expansão de variáveis pelo shell):")
	fmt.Println("    Dockerfile:")
	fmt.Println("      ENV PORT=8080")
	fmt.Println("      ENV HTTP_ACCEPTED_CODES=200,404")
	fmt.Println("      ENV BODY_CHECK_STRING=\"OK\"")
	fmt.Println("      HEALTHCHECK --interval=5s --timeout=3s CMD [\"./httpcheck\", \"-u\", \"http://localhost:${PORT}/api/v1/health\", \"-c\", \"${HTTP_ACCEPTED_CODES}\", \"-b\", \"${BODY_CHECK_STRING}\"]")
	fmt.Println()
	fmt.Println("  Sem Shell (leitura direta das variáveis pelo programa):")
	fmt.Println("    Dockerfile:")
	fmt.Println("      ENV URL_TO_CHECK=http://localhost:8080/api/v1/health")
	fmt.Println("      ENV ACCEPTED_STATUS_CODES=200,404")
	fmt.Println("      ENV RESPONSE_BODY_CONTAINS=\"OK\"")
	fmt.Println("      HEALTHCHECK --interval=5s --timeout=3s CMD [\"./httpcheck\", \"-U\", \"URL_TO_CHECK\", \"-C\", \"ACCEPTED_STATUS_CODES\", \"-B\", \"RESPONSE_BODY_CONTAINS\"]")
	fmt.Println()
	fmt.Println("Com timeout e método HEAD:")
	fmt.Println("  ./httpcheck -u http://localhost:8080/health -c 200 -t 10 -m HEAD")
	fmt.Println()
	fmt.Println("Com verificação do corpo via variável de ambiente (método GET implícito), ignorando códigos:")
	fmt.Println("  ./httpcheck -u http://localhost:8080/status -B \"RESPONSE_BODY_CONTAINS\"")
	fmt.Println()
	fmt.Println("Com verificação do corpo e códigos:")
	fmt.Println("  ./httpcheck -u http://localhost:8080/status -b \"OK\" -c 200")
}
