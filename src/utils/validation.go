package utils

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
	"fmt"
	"api_compiladores/src/models"
)

var (
	// Expresión regular mejorada para validar que el campo Clave_Cliente sea un número entero positivo
	identRegexNumeric = regexp.MustCompile(`^[1-9][0-9]*$`)
	
	// Expresión regular mejorada para validar nombres (letras, acentos, ñ, espacios únicos)
	identRegexNombre = regexp.MustCompile(`^[a-zA-ZáéíóúÁÉÍÓÚñÑüÜ]+(\s[a-zA-ZáéíóúÁÉÍÓÚñÑüÜ]+)*$`)
	
	// Expresión regular para validar números de celular de Chiapas
	identRegexCelular = regexp.MustCompile(`^(91[6-9]|93[24]|96[1-8]|99[24])\d{7}$`)
	
	// Expresión regular mejorada para validar emails con más dominios
	identRegexEmail = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?@(gmail\.com|hotmail\.com|yahoo\.com|outlook\.com|live\.com|icloud\.com|protonmail\.com|aol\.com|msn\.com|gmx\.com|ymail\.com|me\.com|mail\.com|zoho\.com|edu\.mx|edu\.com|edu\.org|institucional\.edu\.mx|unach\.mx|unicach\.mx)$`)
	
	// Patrones peligrosos que podrían indicar ataques
	sqlInjectionPattern = regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|script|javascript|<script|onload|onerror|alert\(|confirm\(|prompt\()`)
	
	// Caracteres especiales no permitidos en nombres
	specialCharsPattern = regexp.MustCompile(`[!@#$%^&*()_+=\[\]{};':"\\|,.<>/?~` + "`" + `0-9]`)
	
	// Números de teléfono conocidos como inválidos o de prueba
	invalidPhonePatterns = regexp.MustCompile(`^(0000000000|1111111111|2222222222|3333333333|4444444444|5555555555|6666666666|7777777777|8888888888|9999999999|1234567890|0987654321)$`)
	
	// Emails temporales o desechables comunes
	disposableEmailDomains = regexp.MustCompile(`@(10minutemail|guerrillamail|mailinator|tempmail|throwaway|yopmail|maildrop|trashmail)\.`)
)

func ValidateCliente(cliente *models.Cliente) {
	errores := make(map[string][]string)

	// Validaciones mejoradas para Nombre
	nombre := strings.TrimSpace(cliente.Nombre)
	if nombre == "" {
		errores["Nombre"] = append(errores["Nombre"], "El campo Nombre es obligatorio")
	} else {
		// Verificar longitud de caracteres UTF-8
		if utf8.RuneCountInString(nombre) < 2 {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre debe tener al menos 2 caracteres")
		}
		if utf8.RuneCountInString(nombre) > 100 {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre no puede exceder 100 caracteres")
		}
		
		// Verificar caracteres válidos
		if !identRegexNombre.MatchString(nombre) {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre solo puede contener letras, acentos y un espacio entre palabras")
		}
		
		// Verificar caracteres especiales y números
		if specialCharsPattern.MatchString(nombre) {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre no puede contener números ni caracteres especiales")
		}
		
		// Verificar espacios múltiples
		if strings.Contains(nombre, "  ") {
			errores["Nombre"] = append(errores["Nombre"], "No se permiten espacios múltiples consecutivos")
		}
		
		// Verificar que no empiece o termine con espacio
		if nombre != strings.TrimSpace(nombre) {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre no puede empezar o terminar con espacios")
		}
		
		// Verificar patrones de inyección
		if sqlInjectionPattern.MatchString(nombre) {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre contiene caracteres o patrones no permitidos")
		}
		
		// Verificar que tenga al menos una letra
		hasLetter := false
		for _, r := range nombre {
			if unicode.IsLetter(r) {
				hasLetter = true
				break
			}
		}
		if !hasLetter {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre debe contener al menos una letra")
		}
		
		// Verificar nombres muy cortos tras quitar espacios
		nombreSinEspacios := strings.ReplaceAll(nombre, " ", "")
		if len(nombreSinEspacios) < 2 {
			errores["Nombre"] = append(errores["Nombre"], "El Nombre debe tener al menos 2 letras (sin contar espacios)")
		}
	}

	// Validaciones mejoradas para Celular
	celular := strings.TrimSpace(cliente.Celular)
	if celular == "" {
		errores["Celular"] = append(errores["Celular"], "El campo Celular es obligatorio")
	} else {
		// Verificar longitud exacta
		if len(celular) != 10 {
			if len(celular) < 10 {
				errores["Celular"] = append(errores["Celular"], "El número de celular debe tener exactamente 10 dígitos (faltan dígitos)")
			} else {
				errores["Celular"] = append(errores["Celular"], "El número de celular debe tener exactamente 10 dígitos (demasiados dígitos)")
			}
		}
		
		// Verificar que solo contenga números
		for _, char := range celular {
			if !unicode.IsDigit(char) {
				errores["Celular"] = append(errores["Celular"], "El número de celular solo puede contener dígitos")
				break
			}
		}
		
		// Verificar formato específico de Chiapas
		if !identRegexCelular.MatchString(celular) {
			errores["Celular"] = append(errores["Celular"], "El número debe corresponder a una lada válida de Chiapas (916-919, 932, 934, 961-968, 992, 994)")
		}
		
		// Verificar patrones de números inválidos
		if invalidPhonePatterns.MatchString(celular) {
			errores["Celular"] = append(errores["Celular"], "El número de celular no puede ser un patrón repetitivo o secuencial")
		}
		
		// Verificar que no empiece con 0
		if strings.HasPrefix(celular, "0") {
			errores["Celular"] = append(errores["Celular"], "El número de celular no puede empezar con 0")
		}
		
		// Verificar límites específicos por lada
		if len(celular) == 10 {
			lada := celular[:3]
			switch lada {
			case "916", "917", "918", "919":
				// Tuxtla Gutiérrez y zona metropolitana
				if celular[3] == '0' || celular[3] == '1' {
					errores["Celular"] = append(errores["Celular"], "Formato inválido para la lada "+lada+" de Tuxtla Gutiérrez")
				}
			case "932", "934":
				// Tapachula y Comitán
				if celular[3] == '0' {
					errores["Celular"] = append(errores["Celular"], "Formato inválido para la lada "+lada)
				}
			}
		}
	}

	// Validaciones mejoradas para Email
	email := strings.TrimSpace(strings.ToLower(cliente.Email))
	if email == "" {
		errores["Email"] = append(errores["Email"], "El campo Email es obligatorio")
	} else {
		// Verificar longitud
		if len(email) < 5 {
			errores["Email"] = append(errores["Email"], "El Email debe tener al menos 5 caracteres")
		}
		if len(email) > 254 {
			errores["Email"] = append(errores["Email"], "El Email no puede exceder 254 caracteres (límite RFC)")
		}
		
		// Verificar estructura básica
		parts := strings.Split(email, "@")
		if len(parts) != 2 {
			errores["Email"] = append(errores["Email"], "El Email debe tener exactamente un símbolo @")
		} else {
			localPart := parts[0]
			domainPart := parts[1]
			
			// Validar parte local (antes del @)
			if len(localPart) == 0 {
				errores["Email"] = append(errores["Email"], "La parte antes del @ no puede estar vacía")
			} else if len(localPart) > 64 {
				errores["Email"] = append(errores["Email"], "La parte antes del @ no puede exceder 64 caracteres")
			}
			
			// Verificar que no empiece o termine con punto
			if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
				errores["Email"] = append(errores["Email"], "El Email no puede empezar o terminar con punto antes del @")
			}
			
			// Verificar puntos consecutivos
			if strings.Contains(localPart, "..") {
				errores["Email"] = append(errores["Email"], "El Email no puede tener puntos consecutivos")
			}
			
			// Validar dominio
			if len(domainPart) == 0 {
				errores["Email"] = append(errores["Email"], "La parte después del @ no puede estar vacía")
			} else if len(domainPart) > 253 {
				errores["Email"] = append(errores["Email"], "El dominio no puede exceder 253 caracteres")
			}
		}
		
		// Verificar formato completo con regex
		if !identRegexEmail.MatchString(email) {
			errores["Email"] = append(errores["Email"], "El Email debe usar un dominio permitido (gmail.com, hotmail.com, yahoo.com, outlook.com, institucional.edu.mx, etc.)")
		}
		
		// Verificar emails desechables
		if disposableEmailDomains.MatchString(email) {
			errores["Email"] = append(errores["Email"], "No se permiten emails temporales o desechables")
		}
		
		// Verificar patrones de inyección
		if sqlInjectionPattern.MatchString(email) {
			errores["Email"] = append(errores["Email"], "El Email contiene caracteres o patrones no permitidos")
		}
		
		// Verificar caracteres especiales no permitidos al inicio
		if len(email) > 0 && (email[0] == '.' || email[0] == '-' || email[0] == '_') {
			errores["Email"] = append(errores["Email"], "El Email no puede empezar con punto, guión o guión bajo")
		}
		
		// Validaciones específicas por dominio
		if strings.Contains(email, "@") {
			domain := strings.Split(email, "@")[1]
			switch domain {
			case "gmail.com":
				// Gmail no permite puntos al final de la parte local
				localPart := strings.Split(email, "@")[0]
				if strings.HasSuffix(localPart, ".") {
					errores["Email"] = append(errores["Email"], "Gmail no permite emails que terminen con punto antes del @")
				}
			case "edu.mx", "institucional.edu.mx", "unach.mx", "unicach.mx":
				// Emails institucionales deben tener formato específico
				localPart := strings.Split(email, "@")[0]
				if len(localPart) < 3 {
					errores["Email"] = append(errores["Email"], "Los emails institucionales deben tener al menos 3 caracteres antes del @")
				}
			}
		}
	}

	// Validaciones cruzadas
	if len(errores) == 0 {
		// Verificar consistencia entre nombre y email si ambos son válidos
		if nombre != "" && email != "" {
			nombreSinEspacios := strings.ToLower(strings.ReplaceAll(nombre, " ", ""))
			emailLocal := strings.ToLower(strings.Split(email, "@")[0])
			
			// Si el email parece ser muy diferente al nombre, dar una advertencia suave
			if len(nombreSinEspacios) > 3 && len(emailLocal) > 3 {
				similarity := calculateSimilarity(nombreSinEspacios, emailLocal)
				if similarity < 0.3 && !strings.Contains(emailLocal, nombreSinEspacios[:3]) {
					// Esta es más una advertencia que un error crítico
					// errores["Email"] = append(errores["Email"], "Recomendación: El email no parece corresponder al nombre proporcionado")
				}
			}
		}
	}

	// Asignar errores en el cliente
	if len(errores) > 0 {
		cliente.Errores = errores
	} else {
		cliente.Errores = nil
	}
}

// Función auxiliar para calcular similitud básica entre strings
func calculateSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}
	
	matches := 0
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}
	
	for i := 0; i < minLen; i++ {
		if i < len(s1) && i < len(s2) && s1[i] == s2[i] {
			matches++
		}
	}
	
	return float64(matches) / float64(minLen)
}

// Función adicional para validar múltiples clientes
func ValidateClientes(clientes []*models.Cliente) map[int]map[string][]string {
	todosErrores := make(map[int]map[string][]string)
	
	for i, cliente := range clientes {
		ValidateCliente(cliente)
		if cliente.Errores != nil {
			// Type assertion para convertir interface{} a map[string][]string
			if errores, ok := cliente.Errores.(map[string][]string); ok && len(errores) > 0 {
				todosErrores[i] = errores
			}
		}
	}
	
	return todosErrores
}

// Función para obtener un resumen de errores
func GetErrorSummary(cliente *models.Cliente) string {
	if cliente.Errores == nil {
		return "Sin errores de validación"
	}
	
	// Type assertion para convertir interface{} a map[string][]string
	errores, ok := cliente.Errores.(map[string][]string)
	if !ok || len(errores) == 0 {
		return "Sin errores de validación"
	}
	
	totalErrores := 0
	for _, listaErrores := range errores {
		totalErrores += len(listaErrores)
	}
	
	return fmt.Sprintf("Se encontraron %d errores en %d campos", totalErrores, len(errores))
}