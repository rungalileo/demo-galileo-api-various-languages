package utils;

import com.google.gson.FieldNamingStrategy;

import java.lang.reflect.Field;

public class SnakeCaseNamingStrategy implements FieldNamingStrategy {
    @Override
    public String translateName(Field field) {
        return toSnakeCase(field.getName());
    }

    private String toSnakeCase(String camelCase) {
        StringBuilder result = new StringBuilder();
        result.append(camelCase.charAt(0));
        for (int i = 1; i < camelCase.length(); i++) {
            char c = camelCase.charAt(i);
            if (Character.isUpperCase(c)) {
                result.append('_').append(Character.toLowerCase(c));
            } else {
                result.append(c);
            }
        }
        return result.toString();
    }
}
